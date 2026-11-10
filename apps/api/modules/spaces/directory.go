package spaces

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/FacileStudio/porte/directory"

	"gorm.io/gorm"
)

const (
	firstImportRetry = 5 * time.Second
	maxImportRetry   = 5 * time.Minute
)

// Directory keeps Agenda's spaces equal to Registre's. It hands Registre the
// spaces Agenda held before Registre owned them, then applies every snapshot
// Registre serves.
//
// The import has to land before the first sync. Apply removes members a
// snapshot no longer lists, so a sync against a Registre that has not yet
// seen Agenda's spaces would empty them of everyone with an SSO identity.
// Until Registre answers, Agenda keeps serving the spaces it already has.
type Directory struct {
	orm    *gorm.DB
	issuer string
	client directory.Client
	syncer *directory.Syncer
	logger *slog.Logger
	retry  time.Duration
	extra  func(context.Context, directory.Snapshot) error
}

// NewDirectory builds the sync over Agenda's tables. The client's issuer is
// also the provider porte writes on identity rows, which is how a Registre
// subject finds its Agenda user.
func NewDirectory(orm *gorm.DB, client directory.Client, logger *slog.Logger) *Directory {
	d := &Directory{orm: orm, issuer: client.Issuer, client: client, logger: logger, retry: firstImportRetry}
	d.syncer = &directory.Syncer{Client: client, Apply: d.sync, Logger: logger}
	return d
}

// OnSnapshot runs fn on every snapshot before the spaces are applied.
func (d *Directory) OnSnapshot(fn func(context.Context, directory.Snapshot) error) {
	d.extra = fn
}

func (d *Directory) sync(ctx context.Context, snapshot directory.Snapshot) error {
	if d.extra != nil {
		if err := d.extra(ctx, snapshot); err != nil {
			return err
		}
	}
	return d.Apply(ctx, snapshot)
}

// Handler answers Registre's nudge with 202 and schedules a pull.
func (d *Directory) Handler() http.Handler {
	return d.syncer.Handler()
}

// Run imports until Registre accepts, then syncs until ctx ends. Each refused
// import doubles the wait, up to five minutes, so a Registre that is not
// deployed yet costs one log line every few minutes and never a crash.
func (d *Directory) Run(ctx context.Context) {
	wait := d.retry
	for {
		err := d.importSpaces(ctx)
		if err == nil {
			break
		}
		d.logger.Warn("directory: import failed, retrying", slog.Duration("in", wait), slog.Any("error", err))
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		wait = min(wait*2, maxImportRetry)
	}
	d.syncer.Run(ctx)
}

// importRow is one membership as the import reads it.
type importRow struct {
	SpaceID int64
	Name    string
	Subject string
	Email   string
	Role    string
}

// importSpaces sends every Agenda space and its members to Registre. Agenda
// has no personal spaces, since a personal calendar simply has none, so every
// space is shared. A member carries their subject under this issuer when they
// have one and their email always, which Registre falls back to. Registre
// ignores a repeat, so calling it on every boot is safe.
func (d *Directory) importSpaces(ctx context.Context) error {
	var rows []importRow
	err := d.orm.WithContext(ctx).Raw(`
		SELECT DISTINCT ON (s.id, u.id) s.id AS space_id, s.name, coalesce(i.subject, '') AS subject, u.email, sm.role
		FROM spaces s
		JOIN space_members sm ON sm.space_id = s.id
		JOIN users u ON u.id = sm.user_id
		LEFT JOIN porte_identities i ON i.user_id = u.id AND i.provider = ?
		ORDER BY s.id, u.id
	`, d.issuer).Scan(&rows).Error
	if err != nil {
		return err
	}
	_, err = d.client.Import(ctx, groupImport(rows))
	return err
}

// groupImport folds membership rows, ordered by space, into import spaces.
func groupImport(rows []importRow) []directory.ImportSpace {
	spaces := []directory.ImportSpace{}
	var last int64
	for _, row := range rows {
		if row.SpaceID != last {
			spaces = append(spaces, directory.ImportSpace{Name: row.Name})
			last = row.SpaceID
		}
		current := &spaces[len(spaces)-1]
		current.Members = append(current.Members, directory.ImportMember{
			Subject: row.Subject, Email: row.Email, Role: agendaRole(row.Role),
		})
	}
	return spaces
}

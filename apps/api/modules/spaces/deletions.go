package spaces

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/FacileStudio/porte/directory"

	"gorm.io/gorm"
)

// Deletions erases the people Registre erased. It polls Registre's deletion
// list with the same client credentials as the snapshot sync and accepts
// Registre's signed push on directory.DeletionsPath.
type Deletions struct {
	orm        *gorm.DB
	storageDir string
	erasers    Erasers
	receiver   *directory.Deletions
}

// Erasers are the two halves of erasing a person, and the check that finds
// it undone. Account is the kit store's EraseAccount, which deletes the
// account and what it owns in one transaction; Identity is porte's own rows
// eraser, pg.Store.EraseIdentity; Holds is pg.Store.HoldsIdentity, which lets
// a re-pushed or re-polled notice erase an account that came back.
type Erasers struct {
	Account  func(ctx context.Context, provider, subject string) error
	Identity func(ctx context.Context, provider, subject string) error
	Holds    func(ctx context.Context, provider, subject string) (bool, error)
}

// NewDeletions builds the receiver over Agenda's tables.
func NewDeletions(
	orm *gorm.DB, client directory.Client, erasers Erasers, storageDir string, logger *slog.Logger,
) *Deletions {
	d := &Deletions{orm: orm, storageDir: storageDir, erasers: erasers}
	d.receiver = &directory.Deletions{
		Client:          client,
		OnPersonDeleted: d.ErasePerson,
		Erase:           erasers.Identity,
		Holds:           erasers.Holds,
		State:           deletionState{orm: orm},
		Logger:          logger,
	}
	return d
}

// Run polls until ctx ends. Start it in exactly one replica.
func (d *Deletions) Run(ctx context.Context) {
	d.receiver.Run(ctx)
}

// Handler verifies and applies Registre's signed user.deleted push.
func (d *Deletions) Handler() http.Handler {
	return d.receiver.Handler()
}

// Erased reports whether Agenda already erased the person behind (provider,
// subject), read from the saved receiver state on every call, so sign-in,
// bearer provisioning and the space sync can refuse to give that identity an
// account again: porte.ErasedSubjects.
func (d *Deletions) Erased(ctx context.Context, provider, subject string) (bool, error) {
	return d.receiver.Erased(ctx, provider, subject)
}

// ErasePerson deletes the account behind a Registre identity through the kit
// store, then removes its uploaded avatar once the rows are gone. A subject
// that is already gone is a no-op, so a replayed notice changes nothing.
func (d *Deletions) ErasePerson(ctx context.Context, provider, subject string) error {
	var uploads []string
	err := d.orm.WithContext(ctx).Raw(
		`SELECT coalesce(u.avatar_upload_path, '') FROM users u
		 JOIN porte_identities i ON i.user_id = u.id WHERE i.provider = ? AND i.subject = ?`,
		provider, subject).Scan(&uploads).Error
	if err != nil {
		return err
	}
	if err := d.erasers.Account(ctx, provider, subject); err != nil {
		return err
	}
	for _, upload := range uploads {
		if upload != "" {
			removeUpload(d.storageDir, upload)
		}
	}
	return nil
}

func removeUpload(storageDir, relative string) {
	path := filepath.Join(storageDir, filepath.Clean("/"+relative))
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("failed to remove an erased person's avatar", slog.Any("error", err))
	}
}

// deletionState keeps the receiver's cursor and handled subjects in one row.
type deletionState struct {
	orm *gorm.DB
}

// Load returns the saved state, or nil before the first save.
func (s deletionState) Load(ctx context.Context) ([]byte, error) {
	var rows []struct{ State []byte }
	err := s.orm.WithContext(ctx).Raw(`SELECT state FROM directory_deletion_state WHERE id = 1`).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0].State, nil
}

// Save replaces the saved state.
func (s deletionState) Save(ctx context.Context, state []byte) error {
	return s.orm.WithContext(ctx).Exec(
		`INSERT INTO directory_deletion_state (id, state, updated_at) VALUES (1, ?, now())
		 ON CONFLICT (id) DO UPDATE SET state = EXCLUDED.state, updated_at = now()`, state).Error
}

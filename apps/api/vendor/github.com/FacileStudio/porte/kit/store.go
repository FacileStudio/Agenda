package kit

import (
	"context"
	"database/sql"
	stderrors "errors"
	"log/slog"
	"strings"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
)

// Store is a porte.UserStore over the app's own users table, through
// database/sql. It needs no schema change: it reads porte_identities, which
// the app already has, and writes only the columns Columns names.
//
// Build it with NewStore, which reads the table once and adopts the
// created_at and updated_at columns a GORM-created table leaves without a
// default. Using a Store literal directly is deprecated: it skips that read,
// writes only the columns Columns names, and leaves those timestamps NULL. A
// literal still works, and logs one warning per process the first time it
// inserts or updates an account. Use NewStore.
//
// OnCreate, when set, runs inside the transaction that inserted a new
// account, after the insert, so an app can fill a column the kit does not
// know (a display colour) or refuse the account by returning an error. What
// OnCreate writes wins over what the insert wrote, timestamps included.
// EraseData is EraseAccount's hook for the rows the erased person owns.
// Logger receives NewStore's warning and the literal warning; nil is
// slog.Default().
type Store struct {
	DB        *sql.DB
	Columns   Columns
	OnCreate  func(ctx context.Context, tx *sql.Tx, userID int64) error
	EraseData func(ctx context.Context, tx *sql.Tx, userID int64) error
	Logger    *slog.Logger

	unwritten []string
	built     bool
}

var (
	_ porte.UserStore = (*Store)(nil)
	_ Resolver        = (*Store)(nil)
)

// UpsertFromOIDC matches on (provider, subject) first, falls back to a
// verified email, and creates the account when neither finds anything. It
// follows porte/pg's linking rules: an address the provider did not verify
// never reaches an existing account, and an account that already holds a
// subject of the same provider is not handed to a second one. Every call
// runs under one advisory lock, so two first sign-ins cannot both become
// the first account. A linked account's profile is refreshed: the picture
// always, the email only when verified, the name and locale only when sent.
// A provider matches with or without one trailing slash, the exact spelling
// first, so an app whose rows mix both spellings needs no data migration.
//
// Calling it on a Store literal is deprecated; build the store with NewStore.
func (s *Store) UpsertFromOIDC(ctx context.Context, claims porte.Claims) (int64, error) {
	p, err := profileFrom(claims)
	if err != nil {
		return 0, err
	}
	var userID int64
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		id, resolveErr := s.resolve(ctx, tx, claims, p)
		userID = id
		return resolveErr
	})
	return userID, err
}

func (s *Store) resolve(ctx context.Context, tx *sql.Tx, claims porte.Claims, p profile) (int64, error) {
	c := s.Columns.resolved()
	spellings := providerSpellings(claims.Provider)
	linked, err := scanID(tx.QueryRowContext(ctx, c.linkedSQL(), spellings[0], spellings[1], claims.Subject))
	if err != nil {
		return 0, errors.Internal("failed to resolve the identity", err)
	}
	if linked == 0 {
		linked, err = scanID(tx.QueryRowContext(ctx, c.byEmailSQL(), p.email))
		if err != nil {
			return 0, errors.Internal("failed to look up the account", err)
		}
		if linked == 0 {
			return s.create(ctx, tx, p)
		}
		if err := mayLink(ctx, tx, linked, claims); err != nil {
			return 0, err
		}
	}
	return linked, s.refresh(ctx, tx, linked, p)
}

func mayLink(ctx context.Context, tx *sql.Tx, userID int64, claims porte.Claims) error {
	if !claims.EmailVerified {
		return errors.Conflict("an account with this email already exists and the identity provider did not verify the address")
	}
	var held int64
	spellings := providerSpellings(claims.Provider)
	err := tx.QueryRowContext(ctx, `SELECT count(*) FROM porte_identities WHERE user_id = $1 AND provider IN ($2, $3)`,
		userID, spellings[0], spellings[1]).Scan(&held)
	if err != nil {
		return errors.Internal("failed to read the account's identities", err)
	}
	if held > 0 {
		return errors.Conflict("this account is already linked to another identity from the same provider")
	}
	return nil
}

// providerSpellings is the provider as given, then the same provider with its
// single trailing slash toggled. Apps configured the issuer both ways over
// time, so a stored identity of either spelling is the same provider.
func providerSpellings(provider string) []string {
	if trimmed, ok := strings.CutSuffix(provider, "/"); ok {
		return []string{provider, trimmed}
	}
	return []string{provider, provider + "/"}
}

func (s *Store) create(ctx context.Context, tx *sql.Tx, p profile) (int64, error) {
	c := s.Columns.resolved()
	args := []any{p.email, p.name, p.picture, localeOrDefault(p.locale)}
	if c.Admin != "" {
		var count int64
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM "+quote(c.Table)).Scan(&count); err != nil {
			return 0, errors.Internal("failed to count users", err)
		}
		args = append(args, count == 0)
	}
	s.warnLiteral()
	var userID int64
	if err := tx.QueryRowContext(ctx, c.insertSQL(), args...).Scan(&userID); err != nil {
		return 0, errors.Internal("failed to create the account", err)
	}
	if s.OnCreate != nil {
		if err := s.OnCreate(ctx, tx, userID); err != nil {
			return 0, err
		}
	}
	return userID, nil
}

func (s *Store) refresh(ctx context.Context, tx *sql.Tx, userID int64, p profile) error {
	s.warnLiteral()
	query, args := s.Columns.resolved().updateSQL(userID, p)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Internal("failed to update the account", err)
	}
	return nil
}

// Identity reads the profile behind a user id porte authenticated. A user
// that no longer exists answers Unauthorized: a session that outlived its
// account is not an authenticated request.
func (s *Store) Identity(ctx context.Context, userID int64) (Identity, error) {
	var out Identity
	err := s.DB.QueryRowContext(ctx, s.Columns.resolved().identitySQL(), userID).
		Scan(&out.UserID, &out.Email, &out.Name, &out.PictureURL, &out.IsAdmin)
	switch {
	case stderrors.Is(err, sql.ErrNoRows):
		return Identity{}, errors.Unauthorized("invalid auth token")
	case err != nil:
		return Identity{}, errors.Internal("failed to load the account", err)
	}
	return out, nil
}

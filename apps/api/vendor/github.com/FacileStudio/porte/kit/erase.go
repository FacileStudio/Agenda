package kit

import (
	"context"
	"database/sql"

	"github.com/FacileStudio/tronc/errors"
)

// EraseAccount deletes the account behind a (provider, subject) identity. It
// has the signature of porte/directory's OnPersonDeleted, which runs before
// porte erases its own rows, so the account still resolves through
// porte_identities here.
//
// One transaction, under the lock UpsertFromOIDC takes, runs EraseData for
// the rows the person owns and then deletes the users row; an EraseData
// error rolls both back. The receiver delivers at least once, so an
// identity that resolves to no account is a success. The provider matches
// with or without one trailing slash, as at sign-in.
func (s *Store) EraseAccount(ctx context.Context, provider, subject string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		return s.erase(ctx, tx, provider, subject)
	})
}

func (s *Store) erase(ctx context.Context, tx *sql.Tx, provider, subject string) error {
	c := s.Columns.resolved()
	spellings := providerSpellings(provider)
	userID, err := scanID(tx.QueryRowContext(ctx, c.linkedSQL(), spellings[0], spellings[1], subject))
	if err != nil {
		return errors.Internal("failed to resolve the erased identity", err)
	}
	if userID == 0 {
		return nil
	}
	if s.EraseData != nil {
		if err := s.EraseData(ctx, tx, userID); err != nil {
			return errors.Internal("failed to erase the account's data", err)
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM "+quote(c.Table)+" WHERE "+quote(c.ID)+" = $1", userID); err != nil {
		return errors.Internal("failed to erase the account", err)
	}
	return nil
}

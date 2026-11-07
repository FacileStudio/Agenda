package pg

import (
	"context"
	stderrors "errors"
	"fmt"
)

const resealBatch = 200

type tokenRow struct {
	provider, subject, access, refresh string
}

// ResealIdentities seals every token still stored in the clear, and every
// token sealed under an older key, under the current key. It returns how many
// rows it rewrote.
//
// It is one-shot and idempotent: a row already sealed under the current key is
// left alone, so a second run rewrites nothing. A row it cannot open is left
// alone too, never blanked. It works in keyset batches, and each update is
// guarded on the values it read, so a concurrent Save wins over the reseal.
// It needs a sealer; without one it returns an error.
func (s *IdentityStore) ResealIdentities(ctx context.Context) (int, error) {
	if s.sealer == nil {
		return 0, stderrors.New("porte/pg: reseal needs a token sealer")
	}
	total := 0
	var last tokenRow
	for {
		rows, err := s.tokenBatch(ctx, last)
		if err != nil {
			return total, err
		}
		changed, err := s.resealRows(ctx, rows)
		total += changed
		if err != nil || len(rows) < resealBatch {
			return total, err
		}
		last = rows[len(rows)-1]
	}
}

func (s *IdentityStore) resealRows(ctx context.Context, rows []tokenRow) (int, error) {
	total := 0
	for _, row := range rows {
		changed, err := s.resealRow(ctx, row)
		if err != nil {
			return total, err
		}
		total += changed
	}
	return total, nil
}

func (s *IdentityStore) tokenBatch(ctx context.Context, after tokenRow) ([]tokenRow, error) {
	cursor, err := s.db.QueryContext(ctx, `
		SELECT provider, subject, access_token, refresh_token
		  FROM porte_identities
		 WHERE (provider, subject) > ($1, $2)
		   AND (access_token <> '' OR refresh_token <> '')
		 ORDER BY provider, subject
		 LIMIT $3`, after.provider, after.subject, resealBatch)
	if err != nil {
		return nil, fmt.Errorf("porte/pg: reseal identities: %w", err)
	}
	defer func() { _ = cursor.Close() }()
	var rows []tokenRow
	for cursor.Next() {
		var row tokenRow
		if err := cursor.Scan(&row.provider, &row.subject, &row.access, &row.refresh); err != nil {
			return nil, fmt.Errorf("porte/pg: reseal identities: %w", err)
		}
		rows = append(rows, row)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("porte/pg: reseal identities: %w", err)
	}
	return rows, nil
}

func (s *IdentityStore) resealRow(ctx context.Context, row tokenRow) (int, error) {
	access, accessChanged := s.resealValue(row.access)
	refresh, refreshChanged := s.resealValue(row.refresh)
	if !accessChanged && !refreshChanged {
		return 0, nil
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE porte_identities SET access_token = $3, refresh_token = $4
		 WHERE provider = $1 AND subject = $2 AND access_token = $5 AND refresh_token = $6`,
		row.provider, row.subject, access, refresh, row.access, row.refresh)
	if err != nil {
		return 0, fmt.Errorf("porte/pg: reseal identities: %w", err)
	}
	written, err := result.RowsAffected()
	return int(written), err
}

func (s *IdentityStore) resealValue(stored string) (string, bool) {
	plain, current, err := s.sealer.open(stored)
	if err != nil || current || stored == "" {
		return stored, false
	}
	sealed, err := s.sealer.Seal(plain)
	if err != nil {
		return stored, false
	}
	return sealed, true
}

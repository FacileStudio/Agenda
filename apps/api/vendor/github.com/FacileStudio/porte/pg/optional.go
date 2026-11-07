package pg

import (
	"context"
	"fmt"
	"sync"
)

var optionalTables = []string{"porte_login_codes", "porte_magic_links"}

type tableSet struct {
	mu      sync.Mutex
	present map[string]bool
}

func newTableSet() *tableSet { return &tableSet{} }

// userTables lists the tables holding rows keyed by user_id that a purge or an
// erase must clear: the required ones always, the optional ones only when the
// app's schema has them. Existence is resolved once per Store with
// to_regclass, which follows the search path as the DELETE does. A failed
// lookup is not cached.
func (t *tableSet) userTables(ctx context.Context, q querier, required ...string) ([]string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.present == nil {
		present := make(map[string]bool, len(optionalTables))
		for _, table := range optionalTables {
			var exists bool
			if err := q.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, table).Scan(&exists); err != nil {
				return nil, fmt.Errorf("porte/pg: look up %s: %w", table, err)
			}
			present[table] = exists
		}
		t.present = present
	}
	tables := append([]string(nil), required...)
	for _, table := range optionalTables {
		if t.present[table] {
			tables = append(tables, table)
		}
	}
	return tables, nil
}

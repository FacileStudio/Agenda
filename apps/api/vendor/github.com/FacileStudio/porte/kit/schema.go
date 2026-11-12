package kit

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"strings"
)

const columnsSQL = `SELECT a.attname, a.attnotnull,
	a.atthasdef OR a.attidentity <> '' OR a.attgenerated <> '', t.typcategory = 'D'
	FROM pg_attribute a JOIN pg_type t ON t.oid = a.atttypid
	WHERE a.attrelid = to_regclass($1) AND a.attnum > 0 AND NOT a.attisdropped ORDER BY a.attnum`

type tableColumn struct {
	name      string
	notNull   bool
	filled    bool
	timestamp bool
}

type tableColumns []tableColumn

// NewStore reads the users table's columns once, through the same search_path
// the kit's queries use, and returns store ready to use. It fails when the
// table or a column Columns names does not exist.
//
// An empty CreatedAt or UpdatedAt adopts a timestamp column named created_at
// or updated_at that has no default, identity or generated expression: the
// columns GORM filled at insert time and a SQL-migrated table fills itself.
// It then lists, as Unwritten, the nullable columns with no default that the
// kit's insert leaves out, and logs them once at warn when OnCreate is nil.
func NewStore(ctx context.Context, store Store) (*Store, error) {
	c := store.Columns.resolved()
	table, err := readColumns(ctx, store.DB, c.Table)
	if err != nil {
		return nil, err
	}
	for _, name := range append([]string{c.ID}, c.inserted()...) {
		if !table.has(name) {
			return nil, fmt.Errorf("porte/kit: table %q has no column %q", c.Table, name)
		}
	}
	store.Columns.CreatedAt = fallback(c.CreatedAt, table.adoptable("created_at"))
	store.Columns.UpdatedAt = fallback(c.UpdatedAt, table.adoptable("updated_at"))
	store.unwritten = table.unwritten(append([]string{c.ID}, store.Columns.resolved().inserted()...))
	store.built = true
	store.warnUnwritten(c.Table)
	return &store, nil
}

func (s *Store) warnUnwritten(table string) {
	if len(s.unwritten) == 0 || s.OnCreate != nil {
		return
	}
	s.logger().Warn("porte/kit: columns "+strings.Join(s.unwritten, ", ")+" of "+table+
		" have no default and will be NULL on accounts the kit creates",
		slog.String("table", table), slog.Any("columns", s.unwritten))
}

// Unwritten is the nullable columns, with no default, that the kit's insert
// leaves out: NULL on every account the kit creates unless OnCreate fills
// them. It is empty on a store that NewStore did not build.
func (s *Store) Unwritten() []string {
	return slices.Clone(s.unwritten)
}

func readColumns(ctx context.Context, db *sql.DB, table string) (tableColumns, error) {
	if db == nil {
		return nil, fmt.Errorf("porte/kit: NewStore needs a database")
	}
	rows, err := db.QueryContext(ctx, columnsSQL, quote(table))
	if err != nil {
		return nil, fmt.Errorf("porte/kit: read the columns of %q: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	var out tableColumns
	for rows.Next() {
		var column tableColumn
		if err := rows.Scan(&column.name, &column.notNull, &column.filled, &column.timestamp); err != nil {
			return nil, fmt.Errorf("porte/kit: read the columns of %q: %w", table, err)
		}
		out = append(out, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("porte/kit: read the columns of %q: %w", table, err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("porte/kit: table %q does not exist", table)
	}
	return out, nil
}

func (t tableColumns) has(name string) bool {
	return slices.ContainsFunc(t, func(column tableColumn) bool { return column.name == name })
}

func (t tableColumns) adoptable(name string) string {
	for _, column := range t {
		if column.name == name && column.timestamp && !column.filled {
			return name
		}
	}
	return ""
}

func (t tableColumns) unwritten(written []string) []string {
	var out []string
	for _, column := range t {
		if !column.filled && !column.notNull && !slices.Contains(written, column.name) {
			out = append(out, column.name)
		}
	}
	return out
}

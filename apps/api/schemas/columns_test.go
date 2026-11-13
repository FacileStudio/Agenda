package schemas

import (
	"errors"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/tronc/mcp"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var queryModels = []any{
	&User{}, &AppSetting{}, &Space{}, &SpaceMember{}, &Calendar{}, &CalendarMember{},
	&Event{}, &ExternalVersion{}, &EventAttendee{}, &ForfaitState{},
}

var timeType = reflect.TypeFor[time.Time]()

var errRollBack = errors.New("roll back")

func compatibleTypes(field reflect.Type) []string {
	for field.Kind() == reflect.Pointer {
		field = field.Elem()
	}
	if field == timeType {
		return []string{"timestamp with time zone"}
	}
	switch field.Kind() {
	case reflect.String:
		return []string{"text", "character varying"}
	case reflect.Bool:
		return []string{"boolean"}
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64:
		return []string{"bigint"}
	case reflect.Int32, reflect.Uint32:
		return []string{"integer", "bigint"}
	case reflect.Int16, reflect.Uint16:
		return []string{"smallint", "integer", "bigint"}
	default:
		return nil
	}
}

func migratedColumnTypes(t *testing.T, db *gorm.DB, table string) map[string]string {
	t.Helper()
	var rows []struct {
		ColumnName string
		DataType   string
	}
	err := db.Raw(`SELECT column_name, data_type FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = ?`, table).Scan(&rows).Error
	if err != nil {
		t.Fatalf("read the columns of %s: %v", table, err)
	}
	columns := map[string]string{}
	for _, row := range rows {
		columns[row.ColumnName] = row.DataType
	}
	return columns
}

// TestGORMModelsMatchTheMigratedSchema checks every column a GORM model reads
// or writes exists in the schema the migrations build, with a type the field
// can hold. AutoMigrate no longer adds a missing column, so a field added to a
// struct without a migration fails here instead of at query time.
func TestGORMModelsMatchTheMigratedSchema(t *testing.T) {
	db := openTestDatabase(t)
	for _, model := range queryModels {
		parsed, err := schema.Parse(model, &sync.Map{}, db.NamingStrategy)
		if err != nil {
			t.Fatalf("parse %T: %v", model, err)
		}
		columns := migratedColumnTypes(t, db, parsed.Table)
		if len(columns) == 0 {
			t.Errorf("%T: table %s does not exist", model, parsed.Table)
			continue
		}
		for _, field := range parsed.Fields {
			if field.DBName == "" {
				continue
			}
			got, ok := columns[field.DBName]
			if !ok {
				t.Errorf("%s.%s: no such column", parsed.Table, field.DBName)
				continue
			}
			want := compatibleTypes(field.FieldType)
			if !slices.Contains(want, got) {
				t.Errorf("%s.%s: column is %s, field %s needs one of %v", parsed.Table, field.DBName, got, field.FieldType, want)
			}
		}
	}
}

func tableShape(t *testing.T, db *gorm.DB, table string, withIndexes bool) []string {
	t.Helper()
	query := `SELECT column_name || ' ' || data_type || ' ' || is_nullable || ' ' || coalesce(column_default, '')
		FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = @table`
	if withIndexes {
		query += ` UNION ALL SELECT replace(indexdef, ' ON ' || schemaname || '.', ' ON ')
		FROM pg_indexes WHERE schemaname = current_schema() AND tablename = @table`
	}
	var shape []string
	if err := db.Raw(query+` ORDER BY 1`, map[string]any{"table": table}).Scan(&shape).Error; err != nil {
		t.Fatalf("read the shape of %s: %v", table, err)
	}
	return shape
}

// assertMatchesLibrary builds a library's own DDL in a scratch schema, inside
// a transaction that is rolled back, and compares each table's shape with the
// one the migrations built. Running the DDL over the migrated schema instead
// proves nothing, since its CREATE TABLE IF NOT EXISTS skips a table that
// already exists. A library bump that adds or changes a column, or an index
// when withIndexes is set, makes it fail, and the change then needs a
// numbered migration of its own.
func assertMatchesLibrary(t *testing.T, db *gorm.DB, ddl string, tables []string, withIndexes bool) {
	t.Helper()
	migrated := map[string][]string{}
	for _, table := range tables {
		migrated[table] = tableShape(t, db, table, withIndexes)
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, statement := range []string{`CREATE SCHEMA agenda_library_drift`, `SET LOCAL search_path TO agenda_library_drift`, ddl} {
			if err := tx.Exec(statement).Error; err != nil {
				t.Fatalf("build the library schema: %v", err)
			}
		}
		for _, table := range tables {
			want := tableShape(t, tx, table, withIndexes)
			if len(want) == 0 {
				t.Errorf("the library no longer creates %s", table)
			} else if !slices.Equal(migrated[table], want) {
				t.Errorf("%s drifted from the library:\nmigrations %v\nlibrary    %v", table, migrated[table], want)
			}
		}
		return errRollBack
	})
	if !errors.Is(err, errRollBack) {
		t.Fatalf("transaction: %v", err)
	}
}

// TestLibrarySchemaIsInTheMigrations checks the migrations build mcp_actions
// exactly as tronc/mcp's PendingSchema does, column for column and index for
// index, since Agenda used to execute it at every boot.
func TestLibrarySchemaIsInTheMigrations(t *testing.T) {
	assertMatchesLibrary(t, openTestDatabase(t), mcp.PendingSchema, []string{"mcp_actions"}, true)
}

// TestPorteTablesMatchPortePG compares the columns of the three porte tables
// Agenda keeps a private copy of with porte/pg's own Schema. The copy leaves
// out porte_users and porte_magic_links and points its foreign keys at
// Agenda's users, so only the columns are compared.
func TestPorteTablesMatchPortePG(t *testing.T) {
	tables := []string{"porte_identities", "porte_sessions", "porte_login_codes"}
	assertMatchesLibrary(t, openTestDatabase(t), portepg.Schema, tables, false)
}

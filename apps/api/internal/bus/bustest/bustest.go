// Package bustest reads and sabotages antenne_outbox in tests.
package bustest

import (
	"encoding/json"
	"testing"

	"gorm.io/gorm"
)

// Row is one queued change.
type Row struct {
	Channel string
	Payload json.RawMessage
}

// Rows returns the queued changes in id order.
func Rows(t *testing.T, db *gorm.DB) []Row {
	t.Helper()
	var raw []struct {
		Channel string
		Payload string
	}
	if err := db.Raw(`SELECT channel, payload::text AS payload FROM antenne_outbox ORDER BY id`).Scan(&raw).Error; err != nil {
		t.Fatalf("read the outbox: %v", err)
	}
	rows := make([]Row, len(raw))
	for i, r := range raw {
		rows[i] = Row{Channel: r.Channel, Payload: json.RawMessage(r.Payload)}
	}
	return rows
}

// Channels returns the channel of every queued change, in id order.
func Channels(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	rows := Rows(t, db)
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Channel
	}
	return out
}

// Clear empties the outbox.
func Clear(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`DELETE FROM antenne_outbox`).Error; err != nil {
		t.Fatalf("clear the outbox: %v", err)
	}
}

// RefuseEnqueues makes every insert into the outbox fail until the test ends,
// so a write whose enqueue shares its transaction rolls back with it.
func RefuseEnqueues(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		`CREATE OR REPLACE FUNCTION bustest_refuse() RETURNS trigger LANGUAGE plpgsql AS
		 $$ BEGIN RAISE EXCEPTION 'bustest: outbox insert refused'; END $$`,
		`CREATE OR REPLACE TRIGGER bustest_refuse BEFORE INSERT ON antenne_outbox FOR EACH ROW EXECUTE FUNCTION bustest_refuse()`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("install the refusing trigger: %v", err)
		}
	}
	t.Cleanup(func() {
		db.Exec(`DROP TRIGGER IF EXISTS bustest_refuse ON antenne_outbox`)
		db.Exec(`DROP FUNCTION IF EXISTS bustest_refuse()`)
	})
}

// Count returns the rows of table matching where.
func Count(t *testing.T, db *gorm.DB, table, where string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.Table(table).Where(where, args...).Count(&n).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// AssertQueued clears the outbox, runs write, and checks it queued exactly
// the channels want, in order.
func AssertQueued(t *testing.T, db *gorm.DB, write func() error, want ...string) {
	t.Helper()
	Clear(t, db)
	if err := write(); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := Channels(t, db)
	if len(got) != len(want) {
		t.Fatalf("queued %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("queued %v, want %v", got, want)
		}
	}
}

// AssertRollsBack refuses every enqueue, runs write, and checks the write
// failed, left the outbox empty, and that unchanged still reports true: the
// change and its queued row share one transaction.
func AssertRollsBack(t *testing.T, db *gorm.DB, write func() error, unchanged func() bool) {
	t.Helper()
	Clear(t, db)
	RefuseEnqueues(t, db)
	if err := write(); err == nil {
		t.Fatal("the write succeeded although its enqueue failed")
	}
	if n := len(Rows(t, db)); n != 0 {
		t.Fatalf("%d outbox rows after a rolled-back write", n)
	}
	if !unchanged() {
		t.Fatal("the write committed although its enqueue failed")
	}
}

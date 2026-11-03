package spaces

import (
	"sync"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/tronc/testdb"

	"gorm.io/gorm"
)

const testBootstrap = "docker compose up -d db   # or any PostgreSQL 16"

var (
	testDBOnce sync.Once
	testDB     *gorm.DB
	testDBErr  error
	testDBCfg  = testdb.Config{Prefix: "agenda_spaces_test", Migrate: schemas.Migrate}
)

func openTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	url, configured := testdb.URL()
	if !configured {
		testdb.Announce(testBootstrap)
		t.Skip(testdb.SkipReason(testBootstrap))
	}

	testDBOnce.Do(func() { testDB, testDBErr = testdb.Open(url, testDBCfg) })
	if testDBErr != nil {
		t.Fatalf("test database: %v", testDBErr)
	}
	if err := testdb.Truncate(testDB, testDBCfg); err != nil {
		t.Fatalf("reset test database: %v", err)
	}
	return testDB
}

func newUser(t *testing.T, db *gorm.DB, email string) int64 {
	t.Helper()
	user := schemas.User{Email: email, Name: email}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return user.ID
}

const testIssuer = "https://sso.test"

// ssoUser creates a user who signs in through Registre as subject.
func ssoUser(t *testing.T, db *gorm.DB, email, subject string) int64 {
	t.Helper()
	id := newUser(t, db, email)
	if err := db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)`,
		id, testIssuer, subject).Error; err != nil {
		t.Fatalf("create identity %s: %v", subject, err)
	}
	return id
}

// spaceWith builds an unlinked space and puts each user in it at the given role.
func spaceWith(t *testing.T, db *gorm.DB, name string, roles map[int64]string) int64 {
	t.Helper()
	space := schemas.Space{Name: name}
	if err := db.Create(&space).Error; err != nil {
		t.Fatalf("create space: %v", err)
	}
	for userID, role := range roles {
		if err := db.Create(&schemas.SpaceMember{SpaceID: space.ID, UserID: userID, Role: role}).Error; err != nil {
			t.Fatalf("create membership: %v", err)
		}
	}
	return space.ID
}

// rolesIn maps each member's email to their role in a space.
func rolesIn(t *testing.T, db *gorm.DB, spaceID int64) map[string]string {
	t.Helper()
	var rows []struct{ Email, Role string }
	if err := db.Raw(`SELECT u.email, sm.role FROM space_members sm JOIN users u ON u.id = sm.user_id
		WHERE sm.space_id = ?`, spaceID).Scan(&rows).Error; err != nil {
		t.Fatalf("read members: %v", err)
	}
	roles := map[string]string{}
	for _, row := range rows {
		roles[row.Email] = row.Role
	}
	return roles
}

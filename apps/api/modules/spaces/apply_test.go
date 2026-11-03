package spaces

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"testing"

	"github.com/FacileStudio/porte/directory"

	"gorm.io/gorm"
)

// testDirectory builds a sync that talks to registreURL while matching
// identities under testIssuer, the two being the same URL in production.
func testDirectory(db *gorm.DB, registreURL string) *Directory {
	sync := NewDirectory(db, directory.Client{Issuer: registreURL}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	sync.issuer = testIssuer
	return sync
}

// studioSnapshot is Registre's view of the space Agenda calls "Studio": alice
// promoted to owner, dave known to Agenda by email only, erin never seen, and
// carol gone. A second space, "Atelier", exists only in Registre.
func studioSnapshot() directory.Snapshot {
	return directory.Snapshot{Version: 7, Spaces: []directory.Space{
		{ID: "r-studio", Name: "STUDIO", Members: []directory.Member{
			{Subject: "sub-alice", Email: "alice@x.test", Role: directory.RoleOwner},
			{Subject: "sub-dave", Email: "Dave@x.test", Role: directory.RoleAdmin},
			{Subject: "sub-erin", Email: "erin@x.test", Name: "Erin", Picture: testIssuer + "/avatars/sub-erin", Role: "editor"},
		}},
		{ID: "r-atelier", Name: "Atelier", Members: []directory.Member{
			{Subject: "sub-alice", Email: "alice@x.test", Role: directory.RoleMember},
		}},
	}}
}

// state is everything Apply may write, timestamps included, so two equal
// states mean the second Apply wrote nothing.
func state(t *testing.T, db *gorm.DB) []map[string]any {
	t.Helper()
	var rows []map[string]any
	err := db.Raw(`
		SELECT 'space' AS kind, id::text AS a, name AS b, coalesce(registre_space_id, '') AS c, updated_at::text AS d FROM spaces
		UNION ALL SELECT 'member', space_id::text, user_id::text, role, joined_at::text FROM space_members
		UNION ALL SELECT 'user', id::text, email, name, oidc_picture_url FROM users
		UNION ALL SELECT 'identity', user_id::text, provider, subject, '' FROM porte_identities
		ORDER BY 1, 2, 3, 4`).Scan(&rows).Error
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	return rows
}

func TestApplyFollowsRegistreAndSparesLocalMembers(t *testing.T) {
	db := openTestDatabase(t)
	alice := ssoUser(t, db, "alice@x.test", "sub-alice")
	carol := ssoUser(t, db, "carol@x.test", "sub-carol")
	bob := newUser(t, db, "bob@x.test")
	dave := newUser(t, db, "dave@x.test")
	studio := spaceWith(t, db, "Studio", map[int64]string{alice: "member", carol: "admin", bob: "member"})
	sync := testDirectory(db, "")

	if err := sync.Apply(context.Background(), studioSnapshot()); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	var linked struct{ Name, RegistreSpaceID string }
	db.Raw(`SELECT name, registre_space_id FROM spaces WHERE id = ?`, studio).Scan(&linked)
	if linked.Name != "STUDIO" || linked.RegistreSpaceID != "r-studio" {
		t.Errorf("Studio = %+v, want linked by name to r-studio and renamed", linked)
	}
	want := map[string]string{"alice@x.test": "owner", "bob@x.test": "member", "dave@x.test": "admin", "erin@x.test": "member"}
	if got := rolesIn(t, db, studio); !reflect.DeepEqual(got, want) {
		t.Errorf("Studio members = %v, want %v (carol removed, bob kept, editor mapped to member)", got, want)
	}

	var daveIdentity, erinPictures int64
	db.Raw(`SELECT user_id FROM porte_identities WHERE provider = ? AND subject = 'sub-dave'`, testIssuer).Scan(&daveIdentity)
	db.Raw(`SELECT count(*) FROM users u JOIN porte_identities i ON i.user_id = u.id
		WHERE i.subject = 'sub-erin' AND u.oidc_picture_url = ? AND u.name = 'Erin'`, testIssuer+"/avatars/sub-erin").Scan(&erinPictures)
	if daveIdentity != dave {
		t.Errorf("dave's identity points at user %d, want %d: matching by email must link the existing account", daveIdentity, dave)
	}
	if erinPictures != 1 {
		t.Error("erin was not pre-provisioned with her name, picture and identity")
	}

	var atelier int64
	db.Raw(`SELECT id FROM spaces WHERE registre_space_id = 'r-atelier'`).Scan(&atelier)
	if got := rolesIn(t, db, atelier); !reflect.DeepEqual(got, map[string]string{"alice@x.test": "member"}) {
		t.Errorf("Atelier members = %v, want alice alone", got)
	}

	before := state(t, db)
	if err := sync.Apply(context.Background(), studioSnapshot()); err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	if after := state(t, db); !reflect.DeepEqual(before, after) {
		t.Errorf("a second Apply of the same snapshot changed something:\nbefore %v\nafter  %v", before, after)
	}
}

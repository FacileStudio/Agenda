package schemas

import (
	"fmt"
	"sync"
	"testing"

	"github.com/FacileStudio/tronc/testdb"
	"gorm.io/gorm"
)

const testBootstrap = "docker compose up -d db   # or any PostgreSQL 16"

var (
	testDBOnce sync.Once
	testDB     *gorm.DB
	testDBErr  error
	testDBCfg  = testdb.Config{Prefix: "agenda_test", Migrate: Migrate}
)

// openTestDatabase hands the test a migrated Postgres, emptied per case. Postgres and not
// SQLite: these tests assert what the shipped SQL does, and SQLite would build a different
// schema from the struct tags and then agree with itself.
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

func TestAvatarPrecedence(t *testing.T) {
	const porte = "https://porte.facile.studio/media/user-avatars/x.png"

	cases := []struct {
		name       string
		user       User
		wantURL    string
		wantOrigin string
	}{
		{"Porte photo wins over an upload", User{OIDCPictureURL: porte, AvatarUploadPath: "avatars/user-3-1.png"}, porte, "oidc"},
		{"upload is the fallback", User{AvatarUploadPath: "avatars/user-3-1.png"}, "/files/avatars/user-3-1.png", "upload"},
		{"only Porte", User{OIDCPictureURL: porte}, porte, "oidc"},
		{"neither, so the client draws initials", User{}, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.user.Avatar(); got != tc.wantURL {
				t.Errorf("Avatar() = %q, want %q", got, tc.wantURL)
			}
			if got := tc.user.AvatarOrigin(); got != tc.wantOrigin {
				t.Errorf("AvatarOrigin() = %q, want %q", got, tc.wantOrigin)
			}
		})
	}
}

// The member joins in spaces and calendars read the avatar in SQL rather than loading the
// row, so the two spellings of the same rule have to agree. This is the test that fails
// when someone edits one and forgets the other.
func TestAvatarSelectExprMatchesAvatar(t *testing.T) {
	orm := openTestDatabase(t)

	users := []User{
		{Email: "both@example.com", OIDCPictureURL: "https://porte.facile.studio/media/user-avatars/a.png", AvatarUploadPath: "avatars/user-1-1.png"},
		{Email: "upload@example.com", AvatarUploadPath: "avatars/user-2-1.png"},
		{Email: "oidc@example.com", OIDCPictureURL: "https://porte.facile.studio/media/user-avatars/b.png"},
		{Email: "neither@example.com"},
	}
	for i := range users {
		if err := orm.Create(&users[i]).Error; err != nil {
			t.Fatalf("create %s: %v", users[i].Email, err)
		}
	}

	for _, want := range users {
		var got string
		if err := orm.Raw(
			fmt.Sprintf(`SELECT %s FROM users u WHERE u.id = ?`, AvatarSelectExpr),
			want.ID).Scan(&got).Error; err != nil {
			t.Fatalf("select for %s: %v", want.Email, err)
		}
		if got != want.Avatar() {
			t.Errorf("%s: SQL gave %q, Avatar() gave %q", want.Email, got, want.Avatar())
		}
	}
}

// Row 2 is the reason this test exists: an uploaded avatar with avatar_source EMPTY,
// because it predates that column. A backfill keyed on avatar_source = 'upload' drops its
// picture without a word. Row 5 is the other half: a data: placeholder that the old sync
// stored verbatim, which under the new rule would masquerade as an SSO photo and suppress
// the upload fallback for good.
// TestBackfillAvatarSources runs the avatar backfill over every shape a row can
// take. The row that carries both sources keeps its file and still renders the
// Porte photo (SSO wins), and the placeholder is gone so an upload can serve as
// the fallback again.
func TestBackfillAvatarSources(t *testing.T) {
	orm := openTestDatabase(t)

	rows := []struct {
		email      string
		url        string
		source     string
		oidc       string
		wantUpload string
		wantOIDC   string
	}{
		{"oidc-copy@example.com", "/files/avatars/oidc-1-178006.png", "oidc", "https://porte.facile.studio/media/user-avatars/a.png", "", "https://porte.facile.studio/media/user-avatars/a.png"},
		{"legacy-upload@example.com", "/files/avatars/user-2-177802.jpg", "", "", "avatars/user-2-177802.jpg", ""},
		{"upload-and-sso@example.com", "/files/avatars/user-3-178096.jpg", "upload", "https://porte.facile.studio/media/user-avatars/b.jpeg", "avatars/user-3-178096.jpg", "https://porte.facile.studio/media/user-avatars/b.jpeg"},
		{"no-avatar@example.com", "", "", "", "", ""},
		{"placeholder@example.com", "", "", "data:image/svg+xml;base64,PHN2Zz48L3N2Zz4=", "", ""},
	}
	for _, row := range rows {
		if err := orm.Exec(
			`INSERT INTO users (email, password_hash, avatar_url, avatar_source, oidc_picture_url) VALUES (?, 'hash', ?, ?, ?)`,
			row.email, row.url, row.source, row.oidc).Error; err != nil {
			t.Fatalf("insert %s: %v", row.email, err)
		}
	}

	if err := backfillAvatarSources(orm); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	for _, row := range rows {
		var got User
		if err := orm.Where("email = ?", row.email).First(&got).Error; err != nil {
			t.Fatalf("read %s: %v", row.email, err)
		}
		if got.AvatarUploadPath != row.wantUpload {
			t.Errorf("%s: avatar_upload_path = %q, want %q", row.email, got.AvatarUploadPath, row.wantUpload)
		}
		if got.OIDCPictureURL != row.wantOIDC {
			t.Errorf("%s: oidc_picture_url = %q, want %q", row.email, got.OIDCPictureURL, row.wantOIDC)
		}
	}

	var both User
	if err := orm.Where("email = ?", "upload-and-sso@example.com").First(&both).Error; err != nil {
		t.Fatalf("read both: %v", err)
	}
	if both.Avatar() != "https://porte.facile.studio/media/user-avatars/b.jpeg" {
		t.Errorf("SSO photo should win, got %q", both.Avatar())
	}

	var placeholder User
	if err := orm.Where("email = ?", "placeholder@example.com").First(&placeholder).Error; err != nil {
		t.Fatalf("read placeholder: %v", err)
	}
	if placeholder.AvatarOrigin() != "" {
		t.Errorf("a data: placeholder should not count as an SSO photo, got origin %q", placeholder.AvatarOrigin())
	}
}

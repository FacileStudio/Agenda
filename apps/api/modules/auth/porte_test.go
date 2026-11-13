package auth

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/testdb"

	"gorm.io/gorm"
)

const (
	testBootstrap = "docker compose up -d db   # or any PostgreSQL 16"
	testIssuer    = "https://sso.test"
)

var (
	testDBOnce sync.Once
	testDB     *gorm.DB
	testDBErr  error
	testDBCfg  = testdb.Config{Prefix: "agenda_auth_test", Migrate: schemas.Migrate}
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

func testStore(t *testing.T, db *gorm.DB) *kit.Store {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	store, err := NewStore(context.Background(), sqlDB, nil)
	if err != nil {
		t.Fatalf("user store: %v", err)
	}
	return store
}

func claims(subject, email string, verified bool) porte.Claims {
	return porte.Claims{Provider: testIssuer, Subject: subject, Email: email, EmailVerified: verified, Locale: "de"}
}

func link(t *testing.T, db *gorm.DB, userID int64, subject string) {
	t.Helper()
	err := db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)`, userID, testIssuer, subject).Error
	if err != nil {
		t.Fatalf("link identity: %v", err)
	}
}

// An account created before Registre owned sign-in has no federated identity.
// Its owner's first Facile sign-in must land on that same row, keep its data,
// and then resolve by subject.
func TestFirstFacileSignInLinksAnExistingAccountByVerifiedEmail(t *testing.T) {
	db := openTestDatabase(t)
	store := testStore(t, db)
	ctx := context.Background()

	existing := schemas.User{Email: "camille@facile.studio", Name: "Camille"}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	id, err := store.UpsertFromOIDC(ctx, claims("sub-1", "Camille@Facile.Studio", true))
	if err != nil {
		t.Fatalf("first sign-in: %v", err)
	}
	if id != existing.ID {
		t.Fatalf("first sign-in created account %d, want the existing %d", id, existing.ID)
	}
	link(t, db, id, "sub-1")

	again, err := store.UpsertFromOIDC(ctx, claims("sub-1", "camille@facile.studio", true))
	if err != nil || again != existing.ID {
		t.Fatalf("second sign-in = %d, %v; want %d", again, err, existing.ID)
	}

	var user schemas.User
	if err := db.First(&user, id).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if user.Locale != "de" {
		t.Errorf("locale = %q, want the sign-in's de", user.Locale)
	}
}

func TestAnUnverifiedAddressNeverLinksToAnExistingAccount(t *testing.T) {
	db := openTestDatabase(t)
	store := testStore(t, db)

	if err := db.Create(&schemas.User{Email: "noah@facile.studio", Name: "Noah"}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := store.UpsertFromOIDC(context.Background(), claims("sub-2", "noah@facile.studio", false)); err == nil {
		t.Fatal("an unverified address linked to an existing account")
	}
}

func TestARecycledAddressIsNotHandedToANewSubject(t *testing.T) {
	db := openTestDatabase(t)
	store := testStore(t, db)

	owner := schemas.User{Email: "lea@facile.studio", Name: "Lea"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	link(t, db, owner.ID, "sub-owner")

	if _, err := store.UpsertFromOIDC(context.Background(), claims("sub-newcomer", "lea@facile.studio", true)); err == nil {
		t.Fatal("a second subject of the same issuer took over a linked account")
	}
}

// kit.NewStore reads the users table once and lists the nullable columns with
// no default that the kit's insert leaves out. oidc_subject is the only one,
// and NULL is what old rows hold there too. A store built as a literal lists
// nothing, so this also fails if NewStore stops going through kit.NewStore.
func TestTheUserStoreLeavesOnlyTheOIDCSubjectUnwritten(t *testing.T) {
	db := openTestDatabase(t)
	if got, want := testStore(t, db).Unwritten(), []string{"oidc_subject"}; !slices.Equal(got, want) {
		t.Fatalf("Unwritten() = %v, want %v", got, want)
	}
}

// Old rows hold a value in every users column but oidc_subject. An account the
// kit creates must too, so no reader meets a NULL that GORM never wrote.
func TestASignUpFillsEveryColumnOldRowsHold(t *testing.T) {
	db := openTestDatabase(t)
	store := testStore(t, db)
	id, err := store.UpsertFromOIDC(context.Background(), claims("sub-columns", "columns@facile.studio", true))
	if err != nil {
		t.Fatalf("sign-up: %v", err)
	}
	var columns []string
	err = db.Raw(`SELECT column_name FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'users' ORDER BY ordinal_position`).Scan(&columns).Error
	if err != nil || len(columns) == 0 {
		t.Fatalf("read the users columns: %v", err)
	}
	for _, column := range columns {
		if slices.Contains(store.Unwritten(), column) {
			continue
		}
		var null bool
		if err := db.Raw(`SELECT "`+column+`" IS NULL FROM users WHERE id = ?`, id).Scan(&null).Error; err != nil {
			t.Fatalf("read %s: %v", column, err)
		}
		if null {
			t.Errorf("%s is NULL on a new account", column)
		}
	}
}

func TestASignedUpAccountLoadsAsAUserRow(t *testing.T) {
	db := openTestDatabase(t)
	id, err := testStore(t, db).UpsertFromOIDC(context.Background(), claims("sub-new", "new@facile.studio", true))
	if err != nil {
		t.Fatalf("sign-up: %v", err)
	}
	var user schemas.User
	if err := db.First(&user, id).Error; err != nil {
		t.Fatalf("load the new account: %v", err)
	}
	if user.CreatedAt.IsZero() || time.Since(user.CreatedAt) > time.Minute {
		t.Errorf("created_at = %v, want the sign-up time", user.CreatedAt)
	}
}

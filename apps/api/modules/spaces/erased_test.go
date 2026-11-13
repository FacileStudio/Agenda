package spaces

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/oidc"
	jwtpkg "github.com/FacileStudio/porte/oidc/jwt"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"

	"gorm.io/gorm"
)

var _ porte.ErasedSubjects = (*Deletions)(nil)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// registreErased is a Registre whose deletion list always names sub-gone.
func registreErased() http.RoundTripper {
	return roundTrip(func(r *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		if r.URL.Path != "/api/directory/deletions" {
			recorder.WriteHeader(http.StatusNotFound)
			return recorder.Result(), nil
		}
		recorder.Header().Set("Content-Type", "application/json")
		body := `{"deleted":[{"sub":"sub-gone","deleted_at":"` + time.Now().UTC().Format(time.RFC3339Nano) + `"}]}`
		if _, err := recorder.WriteString(body); err != nil {
			return nil, err
		}
		return recorder.Result(), nil
	})
}

// polledDeletions is the receiver as main.go builds it, after one poll has
// erased sub-gone and kept sub-other.
func polledDeletions(t *testing.T, db *gorm.DB) (*Deletions, int64) {
	t.Helper()
	ssoUser(t, db, "gone@facile.studio", "sub-gone")
	other := ssoUser(t, db, "other@facile.studio", "sub-other")
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	store := portepg.New(sqlDB)
	deletions := NewDeletions(db,
		directory.Client{Issuer: testIssuer, HTTP: &http.Client{Transport: registreErased()}},
		Erasers{Account: accounts(t, sqlDB).EraseAccount, Identity: store.EraseIdentity, Holds: store.HoldsIdentity},
		t.TempDir(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := deletions.receiver.Poll(context.Background()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if n := count(t, db, `SELECT count(*) FROM porte_identities WHERE subject = 'sub-gone'`); n != 0 {
		t.Fatalf("the first poll left sub-gone")
	}
	return deletions, other
}

// A re-polled notice erases an account that came back for an erased subject,
// without a restart, and leaves everybody else alone.
func TestARecreatedAccountIsErasedAgainOnTheNextPoll(t *testing.T) {
	db := openTestDatabase(t)
	deletions, other := polledDeletions(t, db)
	again := ssoUser(t, db, "gone-again@facile.studio", "sub-gone")

	if err := deletions.receiver.Poll(context.Background()); err != nil {
		t.Fatalf("re-poll: %v", err)
	}

	if n := count(t, db, `SELECT count(*) FROM users WHERE id = ?`, again); n != 0 {
		t.Errorf("the re-created account survived the re-poll")
	}
	if n := count(t, db, `SELECT count(*) FROM users WHERE id = ?`, other); n != 1 {
		t.Errorf("another account was touched")
	}
}

// Erased answers from the saved state, for either issuer spelling, and only
// for the subject Agenda erased.
func TestErasedAnswersFromTheSavedState(t *testing.T) {
	db := openTestDatabase(t)
	deletions, _ := polledDeletions(t, db)
	for _, c := range []struct {
		provider, subject string
		want              bool
	}{
		{testIssuer, "sub-gone", true},
		{testIssuer + "/", "sub-gone", true},
		{testIssuer, "sub-other", false},
		{"https://elsewhere.test", "sub-gone", false},
	} {
		got, err := deletions.Erased(context.Background(), c.provider, c.subject)
		if err != nil || got != c.want {
			t.Errorf("Erased(%s, %s) = %v, %v, want %v", c.provider, c.subject, got, err, c.want)
		}
	}
}

// A snapshot that still lists an erased member does not provision them.
func TestTheSpaceSyncSkipsAnErasedMember(t *testing.T) {
	db := openTestDatabase(t)
	deletions, _ := polledDeletions(t, db)
	dir := NewDirectory(db, directory.Client{Issuer: testIssuer}, slog.New(slog.NewTextHandler(io.Discard, nil))).
		WithErased(deletions)
	stale := directory.Snapshot{Spaces: []directory.Space{{
		ID: "space-erased", Name: "Erased", Members: []directory.Member{
			{Subject: "sub-gone", Email: "gone@facile.studio", Role: directory.RoleMember},
		},
	}}}

	if err := dir.Apply(context.Background(), stale); err != nil {
		t.Fatalf("apply: %v", err)
	}

	if n := count(t, db, `SELECT count(*) FROM users WHERE lower(email) = 'gone@facile.studio'`); n != 0 {
		t.Errorf("the space sync re-created the erased member")
	}
}

// newIssuer is a Registre that only answers discovery, enough for oidc.New.
func newIssuer(t *testing.T) string {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 server.URL,
			"authorization_endpoint": server.URL + "/authorize",
			"token_endpoint":         server.URL + "/token",
			"jwks_uri":               server.URL + "/keys",
		})
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// The OIDC kit built as main.go builds it refuses to provision an account for
// an erased subject from a bearer token, and creates nothing.
func TestBearerProvisioningRefusesAnErasedSubject(t *testing.T) {
	db := openTestDatabase(t)
	polledDeletions(t, db)
	issuer := newIssuer(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	store := portepg.New(sqlDB)
	cfg := porte.Config{
		Issuer: issuer, ClientID: "agenda", ClientSecret: "secret",
		RedirectURL: "https://agenda.test/api/auth/oidc/callback", SuccessURL: "https://agenda.test",
	}
	sessions, err := session.New(cfg, session.Deps{Sessions: store.Sessions(), Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	erased := NewDeletions(db, directory.Client{Issuer: issuer}, Erasers{}, t.TempDir(), logger)
	kit, err := oidc.New(context.Background(), cfg, oidc.Deps{
		Users: accounts(t, sqlDB), Identities: store.Identities(), Sessions: sessions,
		Codes: store.LoginCodes(), Logger: logger, Erased: erased,
	})
	if err != nil {
		t.Fatalf("oidc: %v", err)
	}
	before := count(t, db, `SELECT count(*) FROM users`)
	verified := true

	_, err = kit.ProvisionBearer(context.Background(), jwtpkg.Claims{
		Subject: "sub-gone", Email: "gone@facile.studio", Profile: jwtpkg.Profile{EmailVerified: &verified},
	})

	if !stderrors.Is(err, porte.ErrSubjectErased) {
		t.Fatalf("ProvisionBearer = %v, want a refusal wrapping porte.ErrSubjectErased", err)
	}
	if n := count(t, db, `SELECT count(*) FROM users`); n != before {
		t.Errorf("users = %d after a refused provisioning, want %d", n, before)
	}
}

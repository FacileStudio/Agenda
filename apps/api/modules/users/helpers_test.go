package users

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/internal/env"
	"github.com/FacileStudio/Agenda/apps/api/modules/auth"
	"github.com/FacileStudio/Agenda/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/local"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/testdb"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

const testBootstrap = "docker compose up -d db   # or any PostgreSQL 16"

var (
	testDBOnce sync.Once
	testDB     *gorm.DB
	testDBErr  error
	testDBCfg  = testdb.Config{Prefix: "agenda_users_test", Migrate: schemas.Migrate}
)

// openTestDatabase hands the test a migrated Postgres, emptied per case. It is
// the real database because these cases assert what porte's SQL does with the
// identity table, and porte is the thing under test here.
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

// harness is the real router over a real database: porte's stores, its session
// manager, its password kit, and both modules' own RegisterRoutes.
//
// It is assembled rather than mocked because ChangePassword reads the caller's
// session id out of porte.From(ctx) and writes the rotated cookie through the
// ResponseWriter. A test holding a bare context exercises neither and passes
// whatever the handler does, which is how a password endpoint with no
// confirmation survives a green suite.
type harness struct {
	t        *testing.T
	db       *gorm.DB
	router   chi.Router
	sessions *session.Manager
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	db := openTestDatabase(t)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	store := portepg.New(sqlDB)
	users := auth.NewUserStore(db)
	logger := slog.New(slog.DiscardHandler)

	sessions, err := session.New(porte.Config{}, session.Deps{Sessions: store.Sessions(), Logger: logger})
	if err != nil {
		t.Fatalf("session manager: %v", err)
	}
	passwords, err := local.New(local.Config{MinPasswordLength: 8}, local.Deps{
		Users:      users,
		Identities: store.Identities(),
		Sessions:   sessions,
		Logger:     logger,
		Count:      users.CountUsers,
	})
	if err != nil {
		t.Fatalf("password kit: %v", err)
	}

	authService := auth.NewService(db, sessions, passwords, logger)
	userService := NewService(db, t.TempDir(), authService)

	router := chi.NewRouter()
	sessions.Mount(router)
	auth.RegisterRoutes(router, authService, env.Config{})
	RegisterRoutes(router, userService, authService)

	return &harness{t: t, db: db, router: router, sessions: sessions}
}

// account inserts a user row with no credential of any kind. porte owns the
// password, so an account is a row here and nothing in porte_identities until
// somebody sets one — which is the state a federated-only user is in.
func (h *harness) account(email string) int64 {
	h.t.Helper()
	user := schemas.User{Email: email, Name: "Test"}
	if err := h.db.Create(&user).Error; err != nil {
		h.t.Fatalf("create account: %v", err)
	}
	return user.ID
}

// login mints an unlabelled session, which is what signing in produces and
// what a password change is supposed to end everywhere but here.
func (h *harness) login(userID int64) string {
	h.t.Helper()
	token, _, err := h.sessions.Issue(context.Background(), userID, "")
	if err != nil {
		h.t.Fatalf("issue session: %v", err)
	}
	return token
}

// apiToken mints a labelled session. RevokeLogins spares these on purpose, so
// a password change must not break a token wired into a CalDAV client.
func (h *harness) apiToken(userID int64) string {
	h.t.Helper()
	token, _, err := h.sessions.Issue(context.Background(), userID, "CLI")
	if err != nil {
		h.t.Fatalf("issue api token: %v", err)
	}
	return token
}

// do sends a request as a browser does: the session in a cookie and the CSRF
// header beside it. The header is not optional — porte reads the cookie in
// preference to the Authorization header and refuses a cookie-authenticated
// mutating request without it, so a bearer-only test would pass while every
// write 403s in production.
func (h *harness) do(method, path, token string, body any) *httptest.ResponseRecorder {
	h.t.Helper()

	var payload []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			h.t.Fatalf("encode body: %v", err)
		}
		payload = encoded
	}

	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.AddCookie(&http.Cookie{Name: porte.SessionCookieName, Value: token})
		request.Header.Set(porte.CSRFHeaderName, "1")
	}

	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, request)
	return recorder
}

// errorCode reads the code out of tronc's error envelope, which nests it under
// "error" rather than putting it at the top level.
func errorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope from %q: %v", recorder.Body.String(), err)
	}
	return envelope.Error.Code
}

// rotatedToken returns the session token porte wrote into the response cookie,
// or "" when it wrote none.
func rotatedToken(recorder *httptest.ResponseRecorder) string {
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == porte.SessionCookieName && cookie.Value != "" {
			return cookie.Value
		}
	}
	return ""
}

// authenticates reports whether a token still opens an authenticated route.
func (h *harness) authenticates(token string) bool {
	return h.do(http.MethodGet, "/users/me", token, nil).Code == http.StatusOK
}

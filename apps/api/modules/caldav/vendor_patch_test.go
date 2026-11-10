package caldav

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/FacileStudio/Agenda/apps/api/schemas"

	gocaldav "github.com/emersion/go-webdav/caldav"
)

const propfindAll = `<?xml version="1.0" encoding="utf-8"?><d:propfind xmlns:d="DAV:"><d:allprop/></d:propfind>`

const proppatchColor = `<?xml version="1.0" encoding="utf-8"?>
<d:propertyupdate xmlns:d="DAV:" xmlns:a="http://apple.com/ns/ical/">
<d:set><d:prop><a:calendar-color>#FF0000</a:calendar-color></d:prop></d:set>
</d:propertyupdate>`

type davCase struct {
	method, path, body string
	status             int
	contains           []string
}

func patchedServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	db := openTestDatabase(t)
	user := schemas.User{Email: "alice@example.com", Name: "Alice"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	cal := schemas.Calendar{OwnerID: user.ID, Slug: "work", Name: "Work", SyncToken: "ctag-42"}
	if err := db.Create(&cal).Error; err != nil {
		t.Fatalf("create calendar: %v", err)
	}
	handler := &gocaldav.Handler{Backend: NewBackend(db), Prefix: davPrefix}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r.WithContext(withUser(r.Context(), &user)))
	}))
	t.Cleanup(server.Close)
	return server, davPrefix + "/" + user.Email + "/calendars/" + strconv.FormatInt(cal.ID, 10)
}

func runDAV(t *testing.T, server *httptest.Server, c davCase) {
	t.Helper()
	req, err := http.NewRequest(c.method, server.URL+c.path, strings.NewReader(c.body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Depth", "0")
	req.Header.Set("Content-Type", "application/xml")
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", c.method, c.path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	text := string(raw)
	if resp.StatusCode != c.status {
		t.Fatalf("%s %s = %d, want %d: %s", c.method, c.path, resp.StatusCode, c.status, text)
	}
	for _, want := range c.contains {
		if !strings.Contains(text, want) {
			t.Fatalf("%s %s: body lacks %q: %s", c.method, c.path, want, text)
		}
	}
}

func TestVendoredGoWebdavPatches(t *testing.T) {
	server, calPath := patchedServer(t)
	cases := map[string]davCase{
		"root href is the request path": {
			method: "PROPFIND", path: "/dav/", body: propfindAll, status: http.StatusMultiStatus,
			contains: []string{"<href>/dav/</href>"},
		},
		"principal with a trailing slash": {
			method: "PROPFIND", path: "/dav/alice@example.com/", body: propfindAll, status: http.StatusMultiStatus,
			contains: []string{"calendar-home-set", "/dav/alice@example.com/calendars"},
		},
		"home set with a trailing slash": {
			method: "PROPFIND", path: "/dav/alice@example.com/calendars/", body: propfindAll,
			status: http.StatusMultiStatus, contains: []string{"<resourcetype"},
		},
		"calendar exposes cs:getctag": {
			method: "PROPFIND", path: calPath + "/", body: propfindAll, status: http.StatusMultiStatus,
			contains: []string{"http://calendarserver.org/ns/", "getctag", "ctag-42"},
		},
		"proppatch answers 207 with 403": {
			method: "PROPPATCH", path: calPath + "/", body: proppatchColor, status: http.StatusMultiStatus,
			contains: []string{"calendar-color", "403"},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) { runDAV(t, server, c) })
	}
}

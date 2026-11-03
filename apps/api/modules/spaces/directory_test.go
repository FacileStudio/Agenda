package spaces

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/FacileStudio/porte/directory"
)

// fakeRegistre refuses the import with 404 for the first refusals calls, as
// a Registre without the directory deployed does, then accepts it, and
// records every path in the order it was asked.
type fakeRegistre struct {
	mu       sync.Mutex
	refusals int
	calls    []string
	imported []directory.ImportSpace
	synced   chan struct{}
}

func (f *fakeRegistre) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, r.URL.Path)
	if r.URL.Path == "/api/directory/import" {
		if f.refusals > 0 {
			f.refusals--
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body struct{ Spaces []directory.ImportSpace }
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.imported = body.Spaces
	} else if f.synced != nil {
		close(f.synced)
		f.synced = nil
	}
	_ = json.NewEncoder(w).Encode(directory.Snapshot{Version: 1})
}

func TestImportRetriesUntilAcceptedBeforeTheFirstSync(t *testing.T) {
	db := openTestDatabase(t)
	alice := ssoUser(t, db, "alice@x.test", "sub-alice")
	bob := newUser(t, db, "bob@x.test")
	spaceWith(t, db, "Studio", map[int64]string{alice: "owner", bob: "member"})

	synced := make(chan struct{})
	registre := &fakeRegistre{refusals: 2, synced: synced}
	server := httptest.NewServer(registre)
	defer server.Close()
	sync := testDirectory(db, server.URL)
	sync.retry = time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sync.Run(ctx)
	select {
	case <-synced:
	case <-time.After(5 * time.Second):
		t.Fatal("no snapshot was pulled")
	}
	cancel()

	registre.mu.Lock()
	defer registre.mu.Unlock()
	want := []string{"/api/directory/import", "/api/directory/import", "/api/directory/import", "/api/directory/snapshot"}
	if len(registre.calls) < len(want) || !equalPrefix(registre.calls, want) {
		t.Fatalf("calls = %v, want three imports (two refused) before the first snapshot", registre.calls)
	}
	if len(registre.imported) != 1 || len(registre.imported[0].Members) != 2 {
		t.Fatalf("imported = %+v, want Studio with both members", registre.imported)
	}
	members := registre.imported[0].Members
	if members[0].Subject != "sub-alice" || members[0].Role != "owner" || members[1].Email != "bob@x.test" {
		t.Errorf("members = %+v, want alice by subject as owner and bob by email", members)
	}
}

func equalPrefix(got, want []string) bool {
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

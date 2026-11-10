package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

const refusalBody = `{"error":{"code":"rate_limited","message":"too many requests, try again later"}}` + "\n"

func limitedHandler(requests int, window time.Duration) http.Handler {
	return RateLimit(requests, window)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
}

func hit(handler http.Handler, remoteAddr string) *httptest.ResponseRecorder {
	request := httptest.NewRequest("PROPFIND", "/dav/", nil)
	request.RemoteAddr = remoteAddr
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestRateLimitAdmitsTheBudgetThenRefusesWithTheEnvelope(t *testing.T) {
	handler := limitedHandler(100, time.Minute)
	for i := range 100 {
		if code := hit(handler, "203.0.113.7:5000").Code; code != http.StatusNoContent {
			t.Fatalf("request %d: status %d, want 204", i+1, code)
		}
	}
	refused := hit(handler, "203.0.113.7:6000")
	if refused.Code != http.StatusTooManyRequests {
		t.Fatalf("request 101: status %d, want 429", refused.Code)
	}
	if got := refused.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type %q, want application/json", got)
	}
	retry, err := strconv.Atoi(refused.Header().Get("Retry-After"))
	if err != nil || retry < 1 || retry > 61 {
		t.Fatalf("Retry-After %q, want whole seconds until the window ends, plus one", refused.Header().Get("Retry-After"))
	}
	if got := refused.Body.String(); got != refusalBody {
		t.Fatalf("body %q, want %q", got, refusalBody)
	}
	if code := hit(handler, "198.51.100.2:5000").Code; code != http.StatusNoContent {
		t.Fatalf("another address shared the bucket: status %d", code)
	}
}

func TestRateLimitResetsWhenTheWindowEnds(t *testing.T) {
	handler := limitedHandler(2, 50*time.Millisecond)
	hit(handler, "203.0.113.7:1")
	hit(handler, "203.0.113.7:1")
	if code := hit(handler, "203.0.113.7:1").Code; code != http.StatusTooManyRequests {
		t.Fatalf("third request in the window: status %d, want 429", code)
	}
	time.Sleep(60 * time.Millisecond)
	if code := hit(handler, "203.0.113.7:1").Code; code != http.StatusNoContent {
		t.Fatalf("first request of the next window: status %d, want 204", code)
	}
}

func TestRateLimitCountsConcurrentRequestsExactly(t *testing.T) {
	handler := limitedHandler(100, time.Minute)
	var wg sync.WaitGroup
	var mu sync.Mutex
	admitted := 0
	for range 200 {
		wg.Go(func() {
			if hit(handler, "203.0.113.9:4000").Code == http.StatusNoContent {
				mu.Lock()
				admitted++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if admitted != 100 {
		t.Fatalf("admitted %d of 200 concurrent requests, want exactly 100", admitted)
	}
}

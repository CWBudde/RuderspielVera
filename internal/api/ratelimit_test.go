package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// post sends a lobby creation through h with the given forwarding header and remote address.
func post(t *testing.T, h http.Handler, path, xff, remote string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path,
		strings.NewReader(`{"playerName":"Vera","seats":2}`))
	req.RemoteAddr = remote

	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	return rec
}

// TestRateLimit: the burst passes, the next request from the same client is a 429 with
// err.rateLimited and Retry-After; other clients (by X-Forwarded-For) are unaffected; all three
// creation endpoints share the bucket; non-creation endpoints are not limited.
func TestRateLimit(t *testing.T) {
	t.Parallel()

	s, err := New(Options{Decider: SimpleDecider, CreateRate: 3, CreateBurst: 3})
	if err != nil {
		t.Fatal(err)
	}

	handler := s.Handler(nil)

	for i := range 2 {
		if rec := post(t, handler, "/api/lobbies", "203.0.113.7, 10.0.0.1", "10.0.0.1:1234"); rec.Code != http.StatusCreated {
			t.Fatalf("request %d: %d %s", i, rec.Code, rec.Body)
		}
	}

	// A join with a bad code still costs a token.
	if rec := post(t, handler, "/api/lobbies/ZZZZZZ/join", "203.0.113.7", "10.0.0.2:1"); rec.Code != http.StatusNotFound {
		t.Fatalf("join: %d", rec.Code)
	}

	rec := post(t, handler, "/api/games", "203.0.113.7", "10.0.0.1:1234")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("over the limit: %d", rec.Code)
	}

	var eb ErrorBody

	err = json.Unmarshal(rec.Body.Bytes(), &eb)
	if err != nil || eb.Error != "err.rateLimited" || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("429 body %q (%v), Retry-After %q", rec.Body, err, rec.Header().Get("Retry-After"))
	}

	if rec := post(t, handler, "/api/lobbies", "198.51.100.1", "10.0.0.1:1234"); rec.Code != http.StatusCreated {
		t.Fatalf("other client: %d", rec.Code)
	}

	// Without X-Forwarded-For the remote address counts.
	if rec := post(t, handler, "/api/lobbies", "", "192.0.2.5:999"); rec.Code != http.StatusCreated {
		t.Fatalf("remote addr client: %d", rec.Code)
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/catalog", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.7")

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("catalog limited: %d", rec.Code)
	}
}

// TestIPLimiterRefillAndEvict: tokens come back over time and idle buckets are dropped.
func TestIPLimiterRefillAndEvict(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex

	now := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()

		return now
	}
	advance := func(d time.Duration) {
		mu.Lock()
		now = now.Add(d)
		mu.Unlock()
	}

	lim := newIPLimiter(1, 2, clock) // 1 token/s, burst 2

	for _, ip := range []string{"a", "a", "b"} {
		if ok, _ := lim.allow(ip); !ok {
			t.Fatalf("%s denied within burst", ip)
		}
	}

	ok, wait := lim.allow("a")
	if ok || wait <= 0 || wait > time.Second {
		t.Fatalf("a over burst: ok=%v wait=%v", ok, wait)
	}

	advance(time.Second)

	if ok, _ := lim.allow("a"); !ok {
		t.Fatal("a not refilled after 1s")
	}

	advance(10 * time.Second)
	lim.allow("c")

	if n := lim.size(); n != 1 {
		t.Fatalf("buckets after idle sweep = %d, want 1 (only c)", n)
	}
}

// TestFileStoreDelete: Delete removes the game file and ignores a missing one.
func TestFileStoreDelete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fs := fileStore{dir: dir}
	path := filepath.Join(dir, "abcd.json")

	err := os.WriteFile(path, []byte("{}"), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	err = fs.Delete("abcd")
	if err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(path)
	if !os.IsNotExist(err) {
		t.Fatalf("file still there: %v", err)
	}

	err = fs.Delete("abcd")
	if err != nil {
		t.Fatalf("second delete: %v", err)
	}

	err = fs.Delete("../x")
	if err == nil {
		t.Fatal("invalid ID accepted")
	}
}

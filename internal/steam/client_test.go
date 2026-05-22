package steam

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestProfilePath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "steamid64", input: "76561198115468824", want: "profiles/76561198115468824"},
		{name: "vanity", input: "notsprog", want: "id/notsprog"},
		{name: "id url", input: "https://steamcommunity.com/id/notsprog/", want: "id/notsprog"},
		{name: "profile url", input: "https://steamcommunity.com/profiles/76561198115468824/", want: "profiles/76561198115468824"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := profilePath(tt.input)
			if err != nil {
				t.Fatalf("profilePath returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("profilePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProfilePathRejectsUnsupportedURL(t *testing.T) {
	_, err := profilePath("https://example.com/profiles/76561198115468824/")
	if err == nil {
		t.Fatal("expected unsupported profile URL error")
	}
	var typed *Error
	if !errors.As(err, &typed) || typed.Code != CodeInvalidInput {
		t.Fatalf("expected typed CodeInvalidInput, got %#v", err)
	}
	if typed.HintKey != "hint.invalid_profile_input" {
		t.Fatalf("expected invalid_profile_input hint, got %q", typed.HintKey)
	}
}

func TestRetryAfterSeconds(t *testing.T) {
	got := retryAfter("3")
	if got != 3*time.Second {
		t.Fatalf("retryAfter() = %s, want 3s", got)
	}
}

func TestRetryAfterHTTPDate(t *testing.T) {
	when := time.Now().Add(2 * time.Second).UTC().Format(http.TimeFormat)
	got := retryAfter(when)
	if got <= 0 || got > 3*time.Second {
		t.Fatalf("retryAfter() = %s, want positive delay near 2s", got)
	}
}

func TestStoreAssetURL(t *testing.T) {
	got := storeAssetURL("https://cdn.akamai.steamstatic.com", "steam/apps/264710/${FILENAME}?t=123", "library_600x900.jpg")
	want := "https://cdn.akamai.steamstatic.com/steam/apps/264710/library_600x900.jpg?t=123"
	if got != want {
		t.Fatalf("storeAssetURL() = %q, want %q", got, want)
	}
}

func TestStoreAssetURLPreservesAbsolute(t *testing.T) {
	got := storeAssetURL("https://example.com", "https://other.cdn/${FILENAME}", "x.jpg")
	if got != "https://other.cdn/x.jpg" {
		t.Fatalf("storeAssetURL() did not preserve absolute URL: %q", got)
	}
}

func TestStoreAssetURLUpgradesProtocolRelative(t *testing.T) {
	got := storeAssetURL("https://cdn.akamai.steamstatic.com", "//cdn.akamai.steamstatic.com/${FILENAME}", "x.jpg")
	if got != "https://cdn.akamai.steamstatic.com/x.jpg" {
		t.Fatalf("storeAssetURL() did not upgrade //: %q", got)
	}
}

func TestMediaAssetsFromStoreUsesStoreAssets(t *testing.T) {
	item := &StoreItem{
		Assets: &StoreAssets{
			AssetURLFormat:   "steam/apps/1/${FILENAME}?t=9",
			MainCapsule:      "capsule_616x353.jpg",
			LibraryCapsule2x: "library_600x900_2x.jpg",
			LibraryHero:      "library_hero.jpg",
		},
	}
	got := mediaAssetsFromStore("https://cdn.akamai.steamstatic.com", 1, item)
	if len(got) != 3 {
		t.Fatalf("len(mediaAssetsFromStore()) = %d, want 3", len(got))
	}
	if got[1].Name != "library_600x900_2x" {
		t.Fatalf("second asset name = %q", got[1].Name)
	}
}

// --- HTTP layer tests via httptest -----------------------------------------

// newTestClient returns a Client whose Endpoints all point at the given test
// server, with a fresh Cache so each test is isolated.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c := NewClient("US", "english", 5*time.Second)
	c.Endpoints = Endpoints{
		Store:     srv.URL,
		API:       srv.URL,
		Community: srv.URL,
		CDN:       srv.URL,
	}
	c.Cache = NewCache()
	return c
}

func TestAppDetailsHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"264710":{"success":true,"data":{"name":"Subnautica","steam_appid":264710,"is_free":false}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.AppDetails(264710)
	if err != nil {
		t.Fatalf("AppDetails returned error: %v", err)
	}
	if got.Name != "Subnautica" {
		t.Fatalf("AppDetails().Name = %q", got.Name)
	}

	// Second call should hit cache (no second HTTP roundtrip needed even if
	// the server returned different data, but we don't assert that here).
	got2, err := c.AppDetails(264710)
	if err != nil || got2 != got {
		t.Fatalf("expected cache hit returning same pointer")
	}
}

func TestAppDetailsNotFoundReturnsTypedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"1":{"success":false}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.AppDetails(1)
	if err == nil {
		t.Fatal("expected error for success=false")
	}
	if CodeOf(err) != CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %s", CodeOf(err))
	}
}

func TestRetryOn429HonorsRetryAfter(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		fmt.Fprintln(w, `{"264710":{"success":true,"data":{"name":"Subnautica"}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	start := time.Now()
	if _, err := c.AppDetails(264710); err != nil {
		t.Fatalf("AppDetails returned error: %v", err)
	}
	elapsed := time.Since(start)
	if atomic.LoadInt32(&attempts) != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	// Should sleep ~1s for Retry-After but not double-sleep with the 2s
	// exponential default. Earlier impl waited Retry-After + base = 3s.
	if elapsed > 2*time.Second {
		t.Fatalf("elapsed %s > 2s suggests double-sleep regression", elapsed)
	}
}

func TestRetryOn5xx(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		fmt.Fprintln(w, `{"264710":{"success":true,"data":{"name":"Subnautica"}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := c.AppDetails(264710); err != nil {
		t.Fatalf("expected eventual success, got %v", err)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestRateLimitedExhaustionReturnsTypedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.AppDetails(264710)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if CodeOf(err) != CodeRateLimited {
		t.Fatalf("expected CodeRateLimited, got %s (err=%v)", CodeOf(err), err)
	}
	he, ok := HTTPErrorFromAny(err)
	if !ok || he.Status != 429 {
		t.Fatalf("expected wrapped *HTTPError with 429, got %#v", err)
	}
}

func TestNon2xxReturnsTypedHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.AppDetails(264710)
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	if CodeOf(err) != CodeAccessDenied {
		t.Fatalf("expected CodeAccessDenied, got %s", CodeOf(err))
	}
}

func TestSourceChangedOnInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `not json`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.AppDetails(264710)
	if err == nil {
		t.Fatal("expected JSON decode error")
	}
	if CodeOf(err) != CodeSourceChanged {
		t.Fatalf("expected CodeSourceChanged, got %s", CodeOf(err))
	}
}

func TestRetryAfterTooLongAborts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", strconv.Itoa(60*60))
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	start := time.Now()
	_, err := c.AppDetails(264710)
	if err == nil {
		t.Fatal("expected immediate error for excessive Retry-After")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("client should have bailed quickly, took %s", time.Since(start))
	}
	if CodeOf(err) != CodeRateLimited {
		t.Fatalf("expected CodeRateLimited, got %s", CodeOf(err))
	}
}

func TestMinIntervalThrottles(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		fmt.Fprintln(w, `{"264710":{"success":true,"data":{"name":"x"}}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	c.MinInterval = 200 * time.Millisecond
	c.Cache = NewCache() // ensure no cache-hit short-circuit
	start := time.Now()
	for i := 0; i < 3; i++ {
		c.Cache = NewCache()
		if _, err := c.AppDetails(264710); err != nil {
			t.Fatalf("AppDetails returned error: %v", err)
		}
	}
	elapsed := time.Since(start)
	if elapsed < 300*time.Millisecond {
		t.Fatalf("MinInterval did not throttle; elapsed=%s", elapsed)
	}
	if atomic.LoadInt32(&hits) != 3 {
		t.Fatalf("expected 3 hits, got %d", hits)
	}
}

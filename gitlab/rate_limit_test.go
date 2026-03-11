package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitLabGetRateLimit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("RateLimit-Limit", "2000")
		w.Header().Set("RateLimit-Remaining", "1999")
		w.Header().Set("RateLimit-Reset", "1717243200")
		_, _ = fmt.Fprintf(w, `{"version":"16.0.0","revision":"abc123"}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	rl, err := f.GetRateLimit(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqualInt(t, "Limit", 2000, rl.Limit)
	assertEqualInt(t, "Remaining", 1999, rl.Remaining)
	if rl.Reset.Unix() != 1717243200 {
		t.Errorf("Reset: want unix 1717243200, got %d", rl.Reset.Unix())
	}
}

func TestGitLabGetRateLimitNoHeaders(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"version":"16.0.0","revision":"abc123"}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	rl, err := f.GetRateLimit(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqualInt(t, "Limit", 0, rl.Limit)
	assertEqualInt(t, "Remaining", 0, rl.Remaining)
	if !rl.Reset.IsZero() {
		t.Errorf("Reset: expected zero time, got %v", rl.Reset)
	}
}

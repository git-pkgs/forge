package gitea

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGiteaGetRateLimit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/rate_limit", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"resources": map[string]any{
				"core": map[string]any{
					"limit":     100,
					"remaining": 98,
					"reset":     1717243200,
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	rl, err := f.GetRateLimit(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqualInt(t, "Limit", 100, rl.Limit)
	assertEqualInt(t, "Remaining", 98, rl.Remaining)
	if rl.Reset.Unix() != 1717243200 {
		t.Errorf("Reset: want unix 1717243200, got %d", rl.Reset.Unix())
	}
}

func TestGiteaGetRateLimitNotSupported(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/rate_limit", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.GetRateLimit(context.Background())
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}
}

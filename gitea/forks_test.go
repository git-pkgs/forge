package gitea

import (
	"context"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGiteaListForks(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/forks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"full_name":      "alice/testrepo",
				"name":           "testrepo",
				"html_url":       "https://codeberg.org/alice/testrepo",
				"default_branch": "main",
				"fork":           true,
				"archived":       false,
				"private":        false,
				"owner":          map[string]any{"login": "alice"},
				"created_at":     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
				"updated_at":     time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			},
			{
				"full_name":      "bob/testrepo",
				"name":           "testrepo",
				"html_url":       "https://codeberg.org/bob/testrepo",
				"default_branch": "main",
				"fork":           true,
				"archived":       false,
				"private":        false,
				"owner":          map[string]any{"login": "bob"},
				"created_at":     time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
				"updated_at":     time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	forks, err := f.Repos().ListForks(context.Background(), "testorg", "testrepo", forge.ListForksOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(forks) != 2 {
		t.Fatalf("expected 2 forks, got %d", len(forks))
	}
	assertEqual(t, "forks[0].FullName", "alice/testrepo", forks[0].FullName)
	assertEqual(t, "forks[1].FullName", "bob/testrepo", forks[1].FullName)
}

func TestGiteaListForksNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/nope/forks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Repos().ListForks(context.Background(), "testorg", "nope", forge.ListForksOpts{})
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

package gitlab

import (
	"context"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGitLabListForks(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo/forks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"path_with_namespace": "alice/myrepo",
				"name":                "myrepo",
				"default_branch":      "main",
				"visibility":          "public",
				"namespace":           map[string]any{"path": "alice"},
				"created_at":          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			},
			{
				"path_with_namespace": "bob/myrepo",
				"name":                "myrepo",
				"default_branch":      "main",
				"visibility":          "public",
				"namespace":           map[string]any{"path": "bob"},
				"created_at":          time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	forks, err := f.Repos().ListForks(context.Background(), "mygroup", "myrepo", forge.ListForksOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(forks) != 2 {
		t.Fatalf("expected 2 forks, got %d", len(forks))
	}
	assertEqual(t, "forks[0].FullName", "alice/myrepo", forks[0].FullName)
	assertEqual(t, "forks[1].FullName", "bob/myrepo", forks[1].FullName)
}

func TestGitLabListForksNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fnope/forks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "404 Project Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Repos().ListForks(context.Background(), "mygroup", "nope", forge.ListForksOpts{})
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

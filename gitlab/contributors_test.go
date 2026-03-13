package gitlab

import (
	"context"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitLabListContributors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo/repository/contributors", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"name": "Alice", "email": "alice@example.com", "commits": 42, "additions": 500, "deletions": 100},
			{"name": "Bob", "email": "bob@example.com", "commits": 10, "additions": 200, "deletions": 50},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	contributors, err := f.Repos().ListContributors(context.Background(), "mygroup", "myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(contributors) != 2 {
		t.Fatalf("expected 2 contributors, got %d", len(contributors))
	}
	assertEqual(t, "contributors[0].Name", "Alice", contributors[0].Name)
	assertEqual(t, "contributors[0].Email", "alice@example.com", contributors[0].Email)
	if contributors[0].Contributions != 42 {
		t.Errorf("expected 42 contributions, got %d", contributors[0].Contributions)
	}
	assertEqual(t, "contributors[1].Name", "Bob", contributors[1].Name)
	if contributors[1].Contributions != 10 {
		t.Errorf("expected 10 contributions, got %d", contributors[1].Contributions)
	}
}

func TestGitLabListContributorsNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fnope/repository/contributors", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "404 Project Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Repos().ListContributors(context.Background(), "mygroup", "nope")
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

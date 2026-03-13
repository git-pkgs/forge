package github

import (
	"context"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v82/github"
)

func TestGitHubListContributors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/contributors", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.Contributor{
			{Login: ptr("alice"), Contributions: ptrInt(42), Name: ptr("Alice"), Email: ptr("alice@example.com")},
			{Login: ptr("bob"), Contributions: ptrInt(10)},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	s := &gitHubRepoService{client: c}

	contributors, err := s.ListContributors(context.Background(), "octocat", "hello-world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(contributors) != 2 {
		t.Fatalf("expected 2 contributors, got %d", len(contributors))
	}
	assertEqual(t, "contributors[0].Login", "alice", contributors[0].Login)
	if contributors[0].Contributions != 42 {
		t.Errorf("expected 42 contributions, got %d", contributors[0].Contributions)
	}
	assertEqual(t, "contributors[0].Name", "Alice", contributors[0].Name)
	assertEqual(t, "contributors[0].Email", "alice@example.com", contributors[0].Email)
	assertEqual(t, "contributors[1].Login", "bob", contributors[1].Login)
	if contributors[1].Contributions != 10 {
		t.Errorf("expected 10 contributions, got %d", contributors[1].Contributions)
	}
}

func TestGitHubListContributorsNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/nope/contributors", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	s := &gitHubRepoService{client: c}

	_, err := s.ListContributors(context.Background(), "octocat", "nope")
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

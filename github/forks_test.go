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

func TestGitHubListForks(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/forks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.Repository{
			{FullName: ptr("alice/hello-world"), Name: ptr("hello-world"), Fork: ptrBool(true), Owner: &github.User{Login: ptr("alice")}},
			{FullName: ptr("bob/hello-world"), Name: ptr("hello-world"), Fork: ptrBool(true), Owner: &github.User{Login: ptr("bob")}},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	s := &gitHubRepoService{client: c}

	forks, err := s.ListForks(context.Background(), "octocat", "hello-world", forge.ListForksOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(forks) != 2 {
		t.Fatalf("expected 2 forks, got %d", len(forks))
	}
	assertEqual(t, "forks[0].FullName", "alice/hello-world", forks[0].FullName)
	assertEqual(t, "forks[1].FullName", "bob/hello-world", forks[1].FullName)
}

func TestGitHubListForksNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/nope/forks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	s := &gitHubRepoService{client: c}

	_, err := s.ListForks(context.Background(), "octocat", "nope", forge.ListForksOpts{})
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

func TestGitHubListForksWithLimit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/forks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.Repository{
			{FullName: ptr("alice/hello-world"), Name: ptr("hello-world"), Fork: ptrBool(true), Owner: &github.User{Login: ptr("alice")}},
			{FullName: ptr("bob/hello-world"), Name: ptr("hello-world"), Fork: ptrBool(true), Owner: &github.User{Login: ptr("bob")}},
			{FullName: ptr("carol/hello-world"), Name: ptr("hello-world"), Fork: ptrBool(true), Owner: &github.User{Login: ptr("carol")}},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	s := &gitHubRepoService{client: c}

	forks, err := s.ListForks(context.Background(), "octocat", "hello-world", forge.ListForksOpts{Limit: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(forks) != 2 {
		t.Fatalf("expected 2 forks (limit), got %d", len(forks))
	}
}

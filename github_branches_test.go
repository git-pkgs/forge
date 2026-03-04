package forges

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v82/github"
)

func newTestGitHubBranchService(srv *httptest.Server) *gitHubBranchService {
	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	return &gitHubBranchService{client: c}
}

func TestGitHubBranchList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/branches", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{
				"name": "main",
				"commit": {"sha": "abc123"},
				"protected": true
			},
			{
				"name": "feature",
				"commit": {"sha": "def456"},
				"protected": false
			}
		]`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubBranchService(srv)
	branches, err := svc.List(context.Background(), "octocat", "hello", ListBranchOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(branches) != 2 {
		t.Fatalf("expected 2 branches, got %d", len(branches))
	}

	assertEqual(t, "Name[0]", "main", branches[0].Name)
	assertEqual(t, "SHA[0]", "abc123", branches[0].SHA)
	assertEqualBool(t, "Protected[0]", true, branches[0].Protected)
	assertEqual(t, "Name[1]", "feature", branches[1].Name)
	assertEqualBool(t, "Protected[1]", false, branches[1].Protected)
}

func TestGitHubBranchCreate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/git/ref/heads/main", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"ref": "refs/heads/main",
			"object": {"sha": "abc123"}
		}`)
	})
	mux.HandleFunc("POST /api/v3/repos/octocat/hello/git/refs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"ref": "refs/heads/new-branch",
			"object": {"sha": "abc123"}
		}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubBranchService(srv)
	branch, err := svc.Create(context.Background(), "octocat", "hello", "new-branch", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Name", "new-branch", branch.Name)
	assertEqual(t, "SHA", "abc123", branch.SHA)
}

func TestGitHubBranchDelete(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v3/repos/octocat/hello/git/refs/heads/old-branch", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubBranchService(srv)
	err := svc.Delete(context.Background(), "octocat", "hello", "old-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubBranchDeleteNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v3/repos/octocat/hello/git/refs/heads/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message": "Reference does not exist"}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubBranchService(srv)
	err := svc.Delete(context.Background(), "octocat", "hello", "nope")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

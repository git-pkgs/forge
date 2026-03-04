package forges

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v82/github"
)

func newTestGitHubDeployKeyService(srv *httptest.Server) *gitHubDeployKeyService {
	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	return &gitHubDeployKeyService{client: c}
}

func TestGitHubDeployKeyList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{"id": 1, "title": "deploy-key-1", "key": "ssh-rsa AAAA...", "read_only": true},
			{"id": 2, "title": "deploy-key-2", "key": "ssh-ed25519 AAAA...", "read_only": false}
		]`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubDeployKeyService(srv)
	keys, err := svc.List(context.Background(), "octocat", "hello", ListDeployKeyOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	assertEqual(t, "Title[0]", "deploy-key-1", keys[0].Title)
	assertEqualBool(t, "ReadOnly[0]", true, keys[0].ReadOnly)
	assertEqual(t, "Title[1]", "deploy-key-2", keys[1].Title)
	assertEqualBool(t, "ReadOnly[1]", false, keys[1].ReadOnly)
}

func TestGitHubDeployKeyCreate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello/keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": 3, "title": "new-key", "key": "ssh-rsa BBBB...", "read_only": true}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubDeployKeyService(srv)
	key, err := svc.Create(context.Background(), "octocat", "hello", CreateDeployKeyOpts{
		Title:    "new-key",
		Key:      "ssh-rsa BBBB...",
		ReadOnly: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "Title", "new-key", key.Title)
	assertEqualBool(t, "ReadOnly", true, key.ReadOnly)
}

func TestGitHubDeployKeyDelete(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v3/repos/octocat/hello/keys/1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubDeployKeyService(srv)
	err := svc.Delete(context.Background(), "octocat", "hello", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubDeployKeyDeleteNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v3/repos/octocat/hello/keys/999", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message": "Not Found"}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubDeployKeyService(srv)
	err := svc.Delete(context.Background(), "octocat", "hello", 999)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

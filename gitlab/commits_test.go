package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

const (
	gitLabCommitSHA      = "8e8c483db84b4bee98b60c0593521ed34d9990e8"
	gitLabCommitSHAUpper = "8E8C483DB84B4BEE98B60C0593521ED34D9990E8"
)

func TestGitLabResolveCommit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo/repository/commits/v1.0.0", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":       gitLabCommitSHAUpper,
			"short_id": "8e8c483",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	sha, err := f.Commits().ResolveCommit(context.Background(), "mygroup", "myrepo", "v1.0.0")
	if err != nil {
		t.Fatalf("ResolveCommit() error = %v", err)
	}
	if sha != gitLabCommitSHA {
		t.Errorf("ResolveCommit() = %q, want %q", sha, gitLabCommitSHA)
	}
}

func TestGitLabResolveCommitSkipsRequestForFullSHA(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	sha, err := f.Commits().ResolveCommit(context.Background(), "mygroup", "myrepo", gitLabCommitSHAUpper)
	if err != nil {
		t.Fatalf("ResolveCommit() error = %v", err)
	}
	if sha != gitLabCommitSHA {
		t.Errorf("ResolveCommit() = %q, want normalized %q", sha, gitLabCommitSHA)
	}
	if got := requests.Load(); got != 0 {
		t.Errorf("ResolveCommit() made %d requests for a full SHA, want 0", got)
	}
}

func TestGitLabResolveCommitNotFound(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Commits().ResolveCommit(context.Background(), "mygroup", "myrepo", "missing")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("ResolveCommit() error = %v, want ErrNotFound", err)
	}
}

func TestGitLabResolveCommitRejectsShortSHAInResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo/repository/commits/v1.0.0", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "8e8c483"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	if _, err := f.Commits().ResolveCommit(context.Background(), "mygroup", "myrepo", "v1.0.0"); err == nil {
		t.Fatal("ResolveCommit() error = nil, want invalid SHA error")
	}
}

func TestGitLabResolveCommitRequiresArguments(t *testing.T) {
	f := New("https://gitlab.example.com", "", nil)
	if _, err := f.Commits().ResolveCommit(context.Background(), "mygroup", "myrepo", ""); !errors.Is(err, forge.ErrCommitRefRequired) {
		t.Errorf("ResolveCommit() error = %v, want ErrCommitRefRequired", err)
	}
}

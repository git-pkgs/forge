package gitea

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
	giteaCommitSHA      = "8e8c483db84b4bee98b60c0593521ed34d9990e8"
	giteaCommitSHAUpper = "8E8C483DB84B4BEE98B60C0593521ED34D9990E8"
)

func TestGiteaResolveCommit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/git/commits/v1.0.0", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sha": giteaCommitSHAUpper,
			"url": "https://gitea.example.com/testorg/testrepo/commit/" + giteaCommitSHA,
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	sha, err := f.Commits().ResolveCommit(context.Background(), "testorg", "testrepo", "v1.0.0")
	if err != nil {
		t.Fatalf("ResolveCommit() error = %v", err)
	}
	if sha != giteaCommitSHA {
		t.Errorf("ResolveCommit() = %q, want %q", sha, giteaCommitSHA)
	}
}

func TestGiteaResolveCommitSkipsRequestForFullSHA(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	sha, err := f.Commits().ResolveCommit(context.Background(), "testorg", "testrepo", giteaCommitSHAUpper)
	if err != nil {
		t.Fatalf("ResolveCommit() error = %v", err)
	}
	if sha != giteaCommitSHA {
		t.Errorf("ResolveCommit() = %q, want normalized %q", sha, giteaCommitSHA)
	}
	if got := requests.Load(); got != 0 {
		t.Errorf("ResolveCommit() made %d requests for a full SHA, want 0", got)
	}
}

func TestGiteaResolveCommitNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("/", http.NotFound)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Commits().ResolveCommit(context.Background(), "testorg", "testrepo", "missing")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("ResolveCommit() error = %v, want ErrNotFound", err)
	}
}

func TestGiteaResolveCommitRejectsShortSHAInResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/git/commits/v1.0.0", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"sha": "8e8c483"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	if _, err := f.Commits().ResolveCommit(context.Background(), "testorg", "testrepo", "v1.0.0"); err == nil {
		t.Fatal("ResolveCommit() error = nil, want invalid SHA error")
	}
}

func TestGiteaResolveCommitRequiresArguments(t *testing.T) {
	f := New("https://gitea.example.com", "", nil)
	if _, err := f.Commits().ResolveCommit(context.Background(), "", "testrepo", "v1.0.0"); !errors.Is(err, forge.ErrCommitRefRequired) {
		t.Errorf("ResolveCommit() error = %v, want ErrCommitRefRequired", err)
	}
}

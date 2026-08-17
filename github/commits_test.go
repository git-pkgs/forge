package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestCommitResolverResolveCommit(t *testing.T) {
	const responseSHA = "8E8C483DB84B4BEE98B60C0593521ED34D9990E8"
	const wantSHA = "8e8c483db84b4bee98b60c0593521ed34d9990e8"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/actions/checkout/commits/v4.2.1" {
			t.Errorf("path = %q, want commit endpoint", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("Authorization = %q, want bearer token", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Accept") != "application/vnd.github.v3.sha" {
			t.Errorf("Accept = %q, want SHA media type", r.Header.Get("Accept"))
		}
		_, _ = w.Write([]byte(responseSHA))
	}))
	defer srv.Close()

	resolver, err := NewCommitResolverWithBase(srv.URL, "token", srv.Client())
	if err != nil {
		t.Fatalf("NewCommitResolverWithBase: %v", err)
	}
	got, err := resolver.ResolveCommit(context.Background(), "actions", "checkout", "v4.2.1")
	if err != nil {
		t.Fatalf("ResolveCommit: %v", err)
	}
	if got != wantSHA {
		t.Errorf("ResolveCommit() = %q, want %q", got, wantSHA)
	}
}

func TestCommitResolverReturnsFullSHADirectly(t *testing.T) {
	const sha = "8E8C483DB84B4BEE98B60C0593521ED34D9990E8"
	const want = "8e8c483db84b4bee98b60c0593521ed34d9990e8"
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	defer srv.Close()

	resolver, err := NewCommitResolverWithBase(srv.URL, "", srv.Client())
	if err != nil {
		t.Fatalf("NewCommitResolverWithBase: %v", err)
	}
	got, err := resolver.ResolveCommit(context.Background(), "actions", "checkout", sha)
	if err != nil {
		t.Fatalf("ResolveCommit: %v", err)
	}
	if got != want || requests.Load() != 0 {
		t.Errorf("ResolveCommit() = %q with %d requests, want normalized direct SHA", got, requests.Load())
	}
}

func TestCommitResolverErrors(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		defer srv.Close()
		resolver, err := NewCommitResolverWithBase(srv.URL, "", srv.Client())
		if err != nil {
			t.Fatalf("NewCommitResolverWithBase: %v", err)
		}
		_, err = resolver.ResolveCommit(context.Background(), "actions", "checkout", "missing")
		if !errors.Is(err, forge.ErrNotFound) {
			t.Errorf("ResolveCommit() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("empty SHA", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		defer srv.Close()
		resolver, err := NewCommitResolverWithBase(srv.URL, "", srv.Client())
		if err != nil {
			t.Fatalf("NewCommitResolverWithBase: %v", err)
		}
		if _, err := resolver.ResolveCommit(context.Background(), "actions", "checkout", "v4"); err == nil {
			t.Fatal("ResolveCommit() error = nil, want empty SHA error")
		}
	})

	t.Run("invalid SHA", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("deadbeef"))
		}))
		defer srv.Close()
		resolver, err := NewCommitResolverWithBase(srv.URL, "", srv.Client())
		if err != nil {
			t.Fatalf("NewCommitResolverWithBase: %v", err)
		}
		if _, err := resolver.ResolveCommit(context.Background(), "actions", "checkout", "v4"); err == nil {
			t.Fatal("ResolveCommit() error = nil, want invalid SHA error")
		}
	})

	t.Run("invalid base URL", func(t *testing.T) {
		if _, err := NewCommitResolverWithBase("not a URL", "", nil); err == nil {
			t.Fatal("NewCommitResolverWithBase() error = nil, want invalid URL error")
		}
	})

	t.Run("missing arguments", func(t *testing.T) {
		resolver := NewCommitResolver("", nil)
		if _, err := resolver.ResolveCommit(context.Background(), "actions", "checkout", ""); !errors.Is(err, forge.ErrCommitRefRequired) {
			t.Errorf("ResolveCommit() error = %v, want ErrCommitRefRequired", err)
		}
	})
}

func TestGitHubCommitServiceResolveCommit(t *testing.T) {
	const responseSHA = "8E8C483DB84B4BEE98B60C0593521ED34D9990E8"
	const wantSHA = "8e8c483db84b4bee98b60c0593521ed34d9990e8"

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/commits/v4.2.1", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github.v3.sha" {
			t.Errorf("Accept = %q, want SHA media type", r.Header.Get("Accept"))
		}
		_, _ = w.Write([]byte(responseSHA))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := NewWithBase(srv.URL, "test-token", srv.Client())
	sha, err := f.Commits().ResolveCommit(context.Background(), "octocat", "hello-world", "v4.2.1")
	if err != nil {
		t.Fatalf("ResolveCommit() error = %v", err)
	}
	if sha != wantSHA {
		t.Errorf("ResolveCommit() = %q, want %q", sha, wantSHA)
	}
}

func TestGitHubCommitServiceNotFound(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	f := NewWithBase(srv.URL, "test-token", srv.Client())
	_, err := f.Commits().ResolveCommit(context.Background(), "octocat", "hello-world", "missing")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("ResolveCommit() error = %v, want ErrNotFound", err)
	}
}

func TestGitHubCommitServiceSkipsRequestForFullSHA(t *testing.T) {
	const sha = "8E8C483DB84B4BEE98B60C0593521ED34D9990E8"
	const want = "8e8c483db84b4bee98b60c0593521ed34d9990e8"

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	defer srv.Close()

	f := NewWithBase(srv.URL, "test-token", srv.Client())
	got, err := f.Commits().ResolveCommit(context.Background(), "octocat", "hello-world", sha)
	if err != nil {
		t.Fatalf("ResolveCommit() error = %v", err)
	}
	if got != want {
		t.Errorf("ResolveCommit() = %q, want normalized %q", got, want)
	}
	if n := requests.Load(); n != 0 {
		t.Errorf("ResolveCommit() made %d requests for a full SHA, want 0", n)
	}
}

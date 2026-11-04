package gitea

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGiteaGetRepoSubscription(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   bool
	}{
		{name: "watching", status: http.StatusOK, want: true},
		{name: "not watching", status: http.StatusNotFound, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
			mux.HandleFunc("GET /api/v1/repos/alice/repo/subscription", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				if tt.status == http.StatusOK {
					_ = json.NewEncoder(w).Encode(map[string]any{"subscribed": true, "ignored": false})
				}
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()

			f := New(srv.URL, "test-token", nil)
			sub, err := f.Notifications().GetRepoSubscription(context.Background(), "alice", "repo")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertEqualBool(t, "Subscribed", tt.want, sub.Subscribed)
			assertEqualBool(t, "Ignored", false, sub.Ignored)
		})
	}
}

func TestGiteaSetRepoSubscription(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("PUT /api/v1/repos/alice/repo/subscription", func(w http.ResponseWriter, r *http.Request) {
		called = true
		_ = json.NewEncoder(w).Encode(map[string]any{"subscribed": true, "ignored": false})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	sub, err := f.Notifications().SetRepoSubscription(context.Background(), "alice", "repo", forge.SetSubscriptionOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualBool(t, "called", true, called)
	assertEqualBool(t, "Subscribed", true, sub.Subscribed)
}

func TestGiteaSetRepoSubscriptionIgnoredNotSupported(t *testing.T) {
	f := New("https://codeberg.org", "test-token", nil)
	_, err := f.Notifications().SetRepoSubscription(context.Background(), "alice", "repo", forge.SetSubscriptionOpts{Ignored: true})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}
}

func TestGiteaSetRepoSubscriptionNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("PUT /api/v1/repos/alice/missing/subscription", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Notifications().SetRepoSubscription(context.Background(), "alice", "missing", forge.SetSubscriptionOpts{})
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGiteaDeleteRepoSubscription(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("DELETE /api/v1/repos/alice/repo/subscription", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	if err := f.Notifications().DeleteRepoSubscription(context.Background(), "alice", "repo"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualBool(t, "called", true, called)
}

func TestGiteaThreadSubscriptionsNotSupported(t *testing.T) {
	ctx := context.Background()
	n := New("https://codeberg.org", "test-token", nil).Notifications()

	if _, err := n.GetThreadSubscription(ctx, "1"); !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("GetThreadSubscription: expected ErrNotSupported, got %v", err)
	}
	if _, err := n.SetThreadSubscription(ctx, "1", forge.SetSubscriptionOpts{}); !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("SetThreadSubscription: expected ErrNotSupported, got %v", err)
	}
	if err := n.DeleteThreadSubscription(ctx, "1"); !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("DeleteThreadSubscription: expected ErrNotSupported, got %v", err)
	}
}

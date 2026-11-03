package github

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGitHubGetRepoSubscription(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/subscription", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"subscribed": true, "ignored": false})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	sub, err := newTestGitHubNotificationService(srv).GetRepoSubscription(context.Background(), "octocat", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualBool(t, "Subscribed", true, sub.Subscribed)
	assertEqualBool(t, "Ignored", false, sub.Ignored)
}

func TestGitHubGetRepoSubscriptionNotWatching(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/subscription", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	sub, err := newTestGitHubNotificationService(srv).GetRepoSubscription(context.Background(), "octocat", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualBool(t, "Subscribed", false, sub.Subscribed)
	assertEqualBool(t, "Ignored", false, sub.Ignored)
}

func TestGitHubSetRepoSubscription(t *testing.T) {
	tests := []struct {
		name    string
		opts    forge.SetSubscriptionOpts
		wantSub bool
		wantIgn bool
	}{
		{name: "subscribe", opts: forge.SetSubscriptionOpts{}, wantSub: true},
		{name: "ignore", opts: forge.SetSubscriptionOpts{Ignored: true}, wantIgn: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("PUT /api/v3/repos/octocat/hello/subscription", func(w http.ResponseWriter, r *http.Request) {
				var body map[string]bool
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decoding body: %v", err)
					return
				}
				assertEqualBool(t, "body subscribed", tt.wantSub, body["subscribed"])
				assertEqualBool(t, "body ignored", tt.wantIgn, body["ignored"])
				_ = json.NewEncoder(w).Encode(body)
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()

			sub, err := newTestGitHubNotificationService(srv).SetRepoSubscription(context.Background(), "octocat", "hello", tt.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertEqualBool(t, "Subscribed", tt.wantSub, sub.Subscribed)
			assertEqualBool(t, "Ignored", tt.wantIgn, sub.Ignored)
		})
	}
}

func TestGitHubSetRepoSubscriptionNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v3/repos/octocat/missing/subscription", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, err := newTestGitHubNotificationService(srv).SetRepoSubscription(context.Background(), "octocat", "missing", forge.SetSubscriptionOpts{})
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGitHubDeleteRepoSubscription(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v3/repos/octocat/hello/subscription", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	if err := newTestGitHubNotificationService(srv).DeleteRepoSubscription(context.Background(), "octocat", "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualBool(t, "called", true, called)
}

func TestGitHubGetThreadSubscription(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/notifications/threads/42/subscription", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"subscribed": false, "ignored": true})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	sub, err := newTestGitHubNotificationService(srv).GetThreadSubscription(context.Background(), "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualBool(t, "Subscribed", false, sub.Subscribed)
	assertEqualBool(t, "Ignored", true, sub.Ignored)
}

func TestGitHubGetThreadSubscriptionNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/notifications/threads/999/subscription", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, err := newTestGitHubNotificationService(srv).GetThreadSubscription(context.Background(), "999")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGitHubSetThreadSubscription(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v3/notifications/threads/42/subscription", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]bool
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
			return
		}
		assertEqualBool(t, "body subscribed", false, body["subscribed"])
		assertEqualBool(t, "body ignored", true, body["ignored"])
		_ = json.NewEncoder(w).Encode(body)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	sub, err := newTestGitHubNotificationService(srv).SetThreadSubscription(context.Background(), "42", forge.SetSubscriptionOpts{Ignored: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualBool(t, "Subscribed", false, sub.Subscribed)
	assertEqualBool(t, "Ignored", true, sub.Ignored)
}

func TestGitHubDeleteThreadSubscription(t *testing.T) {
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v3/notifications/threads/42/subscription", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	if err := newTestGitHubNotificationService(srv).DeleteThreadSubscription(context.Background(), "42"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualBool(t, "called", true, called)
}

func TestGitHubDeleteThreadSubscriptionNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v3/notifications/threads/999/subscription", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	err := newTestGitHubNotificationService(srv).DeleteThreadSubscription(context.Background(), "999")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

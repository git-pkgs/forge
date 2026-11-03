//go:build !tinygo

package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGitLabGetRepoSubscription(t *testing.T) {
	tests := []struct {
		level       string
		wantSub     bool
		wantIgnored bool
	}{
		{level: "watch", wantSub: true},
		{level: "disabled", wantIgnored: true},
		{level: "global"},
		{level: "participating"},
		{level: "custom"},
	}
	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("GET /api/v4/projects/group%2Frepo/notification_settings", func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprintf(w, `{"level":%q}`, tt.level)
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()

			f := New(srv.URL, "test-token", nil)
			sub, err := f.Notifications().GetRepoSubscription(context.Background(), "group", "repo")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertEqualBool(t, "Subscribed", tt.wantSub, sub.Subscribed)
			assertEqualBool(t, "Ignored", tt.wantIgnored, sub.Ignored)
		})
	}
}

func TestGitLabGetRepoSubscriptionNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/group%2Fmissing/notification_settings", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message":"404 Project Not Found"}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Notifications().GetRepoSubscription(context.Background(), "group", "missing")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// projectLevelHandler records the level sent to the project notification
// settings endpoint and echoes it back.
func projectLevelHandler(t *testing.T, got *string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Level string `json:"level"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
			return
		}
		*got = body.Level
		_, _ = fmt.Fprintf(w, `{"level":%q}`, body.Level)
	}
}

func TestGitLabSetRepoSubscription(t *testing.T) {
	tests := []struct {
		name        string
		opts        forge.SetSubscriptionOpts
		wantLevel   string
		wantSub     bool
		wantIgnored bool
	}{
		{name: "subscribe", opts: forge.SetSubscriptionOpts{}, wantLevel: "watch", wantSub: true},
		{name: "ignore", opts: forge.SetSubscriptionOpts{Ignored: true}, wantLevel: "disabled", wantIgnored: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotLevel string
			mux := http.NewServeMux()
			mux.HandleFunc("PUT /api/v4/projects/group%2Frepo/notification_settings", projectLevelHandler(t, &gotLevel))
			srv := httptest.NewServer(mux)
			defer srv.Close()

			f := New(srv.URL, "test-token", nil)
			sub, err := f.Notifications().SetRepoSubscription(context.Background(), "group", "repo", tt.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertEqual(t, "level sent", tt.wantLevel, gotLevel)
			assertEqualBool(t, "Subscribed", tt.wantSub, sub.Subscribed)
			assertEqualBool(t, "Ignored", tt.wantIgnored, sub.Ignored)
		})
	}
}

func TestGitLabDeleteRepoSubscription(t *testing.T) {
	var gotLevel string
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v4/projects/group%2Frepo/notification_settings", projectLevelHandler(t, &gotLevel))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	if err := f.Notifications().DeleteRepoSubscription(context.Background(), "group", "repo"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "level sent", "global", gotLevel)
}

func TestGitLabThreadSubscriptionsNotSupported(t *testing.T) {
	ctx := context.Background()
	n := New("https://gitlab.com", "test-token", nil).Notifications()

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

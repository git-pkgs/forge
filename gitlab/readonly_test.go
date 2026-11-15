//go:build !tinygo

package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGitLabReadOnlyHTTPClient(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/user", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"username": "jdoe"})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", forge.ReadOnlyHTTPClient(srv.Client()))
	ctx := context.Background()

	u, err := f.GetAuthenticatedUser(ctx)
	if err != nil {
		t.Fatalf("GetAuthenticatedUser: %v", err)
	}
	assertEqual(t, "Login", "jdoe", u.Login)

	if _, err := f.Issues().Create(ctx, "jdoe", "hello", forge.CreateIssueOpts{Title: "x"}); !errors.Is(err, forge.ErrReadOnly) {
		t.Errorf("Issues().Create: err = %v, want ErrReadOnly", err)
	}
	if err := f.Repos().Delete(ctx, "jdoe", "hello"); !errors.Is(err, forge.ErrReadOnly) {
		t.Errorf("Repos().Delete: err = %v, want ErrReadOnly", err)
	}
}

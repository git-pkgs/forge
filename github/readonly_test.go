package github

import (
	"context"
	"encoding/json"
	"errors"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitHubReadOnlyHTTPClient(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/user", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"login": "octocat"})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := NewWithBase(srv.URL, "test-token", forge.ReadOnlyHTTPClient(srv.Client()))
	ctx := context.Background()

	u, err := f.GetAuthenticatedUser(ctx)
	if err != nil {
		t.Fatalf("GetAuthenticatedUser: %v", err)
	}
	assertEqual(t, "Login", "octocat", u.Login)

	if _, err := f.Issues().Create(ctx, "octocat", "hello", forge.CreateIssueOpts{Title: "x"}); !errors.Is(err, forge.ErrReadOnly) {
		t.Errorf("Issues().Create: err = %v, want ErrReadOnly", err)
	}
	if err := f.Repos().Delete(ctx, "octocat", "hello"); !errors.Is(err, forge.ErrReadOnly) {
		t.Errorf("Repos().Delete: err = %v, want ErrReadOnly", err)
	}
}

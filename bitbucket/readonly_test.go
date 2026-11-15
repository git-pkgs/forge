package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestBitbucketReadOnlyHTTPClient(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/user", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"username": "alice"})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	f := New("test-token", forge.ReadOnlyHTTPClient(srv.Client()))
	ctx := context.Background()

	u, err := f.GetAuthenticatedUser(ctx)
	if err != nil {
		t.Fatalf("GetAuthenticatedUser: %v", err)
	}
	assertEqual(t, "Login", "alice", u.Login)

	if _, err := f.Issues().Create(ctx, "alice", "hello", forge.CreateIssueOpts{Title: "x"}); !errors.Is(err, forge.ErrReadOnly) {
		t.Errorf("Issues().Create: err = %v, want ErrReadOnly", err)
	}
	if err := f.Repos().Delete(ctx, "alice", "hello"); !errors.Is(err, forge.ErrReadOnly) {
		t.Errorf("Repos().Delete: err = %v, want ErrReadOnly", err)
	}
}

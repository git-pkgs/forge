package gerrit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestReadOnlyHTTPClient(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /a/accounts/self", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, xssi+`{"username": "alice"}`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "alice:secret", forge.ReadOnlyHTTPClient(srv.Client()))
	ctx := context.Background()

	u, err := f.GetAuthenticatedUser(ctx)
	if err != nil {
		t.Fatalf("GetAuthenticatedUser: %v", err)
	}
	if u.Login != "alice" {
		t.Errorf("Login = %q, want alice", u.Login)
	}

	if err := f.PullRequests().Close(ctx, "org", "repo", 42); !errors.Is(err, forge.ErrReadOnly) {
		t.Errorf("PullRequests().Close: err = %v, want ErrReadOnly", err)
	}
	if err := f.Branches().Delete(ctx, "org", "repo", "topic"); !errors.Is(err, forge.ErrReadOnly) {
		t.Errorf("Branches().Delete: err = %v, want ErrReadOnly", err)
	}
}

package gitea

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGiteaListContributorsNotSupported(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Repos().ListContributors(context.Background(), "testorg", "testrepo")
	if err != forge.ErrNotSupported {
		t.Fatalf("expected forge.ErrNotSupported, got %v", err)
	}
}

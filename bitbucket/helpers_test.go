package bitbucket

import (
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func assertEqual[T ~string](t *testing.T, field, want string, got T) {
	t.Helper()
	if want != string(got) {
		t.Errorf("%s: want %q, got %q", field, want, got)
	}
}

func assertEqualBool(t *testing.T, field string, want, got bool) {
	t.Helper()
	if want != got {
		t.Errorf("%s: want %v, got %v", field, want, got)
	}
}

func assertEqualInt(t *testing.T, field string, want, got int) {
	t.Helper()
	if want != got {
		t.Errorf("%s: want %d, got %d", field, want, got)
	}
}

// newTestForge points the Bitbucket API at a test server running h and
// returns a forge that authenticates with "test-token", along with the
// server's URL.
func newTestForge(t *testing.T, h http.Handler) (forge.Forge, string) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	t.Cleanup(func() { setBitbucketAPI(origAPI) })
	return New("test-token", srv.Client()), srv.URL
}

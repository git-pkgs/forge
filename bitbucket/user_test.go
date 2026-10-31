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

func TestBitbucketGetAuthenticatedUser(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/user", func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "Authorization", "Bearer test-token", r.Header.Get("Authorization"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"username":     "alice",
			"display_name": "Alice Example",
			"links": map[string]any{
				"html":   map[string]string{"href": "https://bitbucket.org/alice/"},
				"avatar": map[string]string{"href": "https://bitbucket.org/account/alice/avatar/"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	f := New("test-token", srv.Client())
	u, err := f.GetAuthenticatedUser(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Login", "alice", u.Login)
	assertEqual(t, "Name", "Alice Example", u.Name)
	assertEqual(t, "Email", "", u.Email)
	assertEqual(t, "AvatarURL", "https://bitbucket.org/account/alice/avatar/", u.AvatarURL)
	assertEqual(t, "HTMLURL", "https://bitbucket.org/alice/", u.HTMLURL)
}

func TestBitbucketGetAuthenticatedUserUnauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/user", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	f := New("", srv.Client())
	_, err := f.GetAuthenticatedUser(context.Background())
	var httpErr *forge.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401 error, got %v", err)
	}
}

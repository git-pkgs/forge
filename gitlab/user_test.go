//go:build !tinygo

package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitLabGetAuthenticatedUser(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/user", func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "PRIVATE-TOKEN", "test-token", r.Header.Get("PRIVATE-TOKEN"))
		_, _ = fmt.Fprint(w, `{
			"id": 1,
			"username": "jdoe",
			"name": "Jane Doe",
			"email": "jdoe@example.com",
			"avatar_url": "https://gitlab.com/uploads/jdoe.png",
			"web_url": "https://gitlab.com/jdoe"
		}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	u, err := f.GetAuthenticatedUser(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Login", "jdoe", u.Login)
	assertEqual(t, "Name", "Jane Doe", u.Name)
	assertEqual(t, "Email", "jdoe@example.com", u.Email)
	assertEqual(t, "AvatarURL", "https://gitlab.com/uploads/jdoe.png", u.AvatarURL)
	assertEqual(t, "HTMLURL", "https://gitlab.com/jdoe", u.HTMLURL)
}

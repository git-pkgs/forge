package gerrit

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAuthenticatedUser(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /a/accounts/self", func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != "alice" || password != "secret" {
			t.Errorf("basic auth = %q, %q, %v; want alice, secret, true", user, password, ok)
		}
		_, _ = fmt.Fprint(w, xssi+`{
			"_account_id": 1000096,
			"name": "Alice Example",
			"email": "alice@example.com",
			"username": "alice"
		}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "alice:secret", srv.Client())
	u, err := f.GetAuthenticatedUser(context.Background())
	if err != nil {
		t.Fatalf("GetAuthenticatedUser: %v", err)
	}
	if u.Login != "alice" || u.Name != "Alice Example" || u.Email != "alice@example.com" {
		t.Fatalf("unexpected user: %+v", u)
	}
}

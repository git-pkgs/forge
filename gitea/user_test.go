package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGiteaGetAuthenticatedUser(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/user", func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "Authorization", "token test-token", r.Header.Get("Authorization"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         1,
			"login":      "alice",
			"full_name":  "Alice Example",
			"email":      "alice@example.com",
			"avatar_url": "https://codeberg.org/avatars/alice",
			"html_url":   "https://codeberg.org/alice",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	u, err := f.GetAuthenticatedUser(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Login", "alice", u.Login)
	assertEqual(t, "Name", "Alice Example", u.Name)
	assertEqual(t, "Email", "alice@example.com", u.Email)
	assertEqual(t, "AvatarURL", "https://codeberg.org/avatars/alice", u.AvatarURL)
	assertEqual(t, "HTMLURL", "https://codeberg.org/alice", u.HTMLURL)
}

func TestGiteaGetAuthenticatedUserUnauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/user", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":""}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "expired-token", nil)
	_, err := f.GetAuthenticatedUser(context.Background())
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	for _, want := range []string{"get authenticated user", "401"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
}

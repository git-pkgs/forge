package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v82/github"
)

func TestGitHubGetAuthenticatedUser(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/user", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"login":      "octocat",
			"name":       "The Octocat",
			"email":      "octocat@github.com",
			"avatar_url": "https://avatars.githubusercontent.com/u/583231",
			"html_url":   "https://github.com/octocat",
			"type":       "User",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	f := &gitHubForge{client: c}

	u, err := f.GetAuthenticatedUser(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Login", "octocat", u.Login)
	assertEqual(t, "Name", "The Octocat", u.Name)
	assertEqual(t, "Email", "octocat@github.com", u.Email)
	assertEqual(t, "AvatarURL", "https://avatars.githubusercontent.com/u/583231", u.AvatarURL)
	assertEqual(t, "HTMLURL", "https://github.com/octocat", u.HTMLURL)
	assertEqualBool(t, "IsOrg", false, u.IsOrg)
}

func TestGitHubGetAuthenticatedUserUnauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/user", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Requires authentication"}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	f := &gitHubForge{client: c}

	if _, err := f.GetAuthenticatedUser(context.Background()); err == nil {
		t.Fatal("expected error for unauthenticated request")
	}
}

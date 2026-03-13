package github

import (
	"context"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v82/github"
)

func newTestGitHubCollaboratorService(srv *httptest.Server) *gitHubCollaboratorService {
	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	return &gitHubCollaboratorService{client: c}
}

func TestGitHubListCollaborators(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/collaborators", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.User{
			{Login: ptr("alice"), Permissions: map[string]bool{"admin": true, "push": true, "pull": true}},
			{Login: ptr("bob"), Permissions: map[string]bool{"admin": false, "push": true, "pull": true}},
			{Login: ptr("carol"), Permissions: map[string]bool{"admin": false, "push": false, "pull": true}},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubCollaboratorService(srv)
	collabs, err := s.List(context.Background(), "octocat", "hello-world", forge.ListCollaboratorOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(collabs) != 3 {
		t.Fatalf("expected 3 collaborators, got %d", len(collabs))
	}
	assertEqual(t, "collabs[0].Login", "alice", collabs[0].Login)
	assertEqual(t, "collabs[0].Permission", "admin", collabs[0].Permission)
	assertEqual(t, "collabs[1].Login", "bob", collabs[1].Login)
	assertEqual(t, "collabs[1].Permission", "write", collabs[1].Permission)
	assertEqual(t, "collabs[2].Login", "carol", collabs[2].Login)
	assertEqual(t, "collabs[2].Permission", "read", collabs[2].Permission)
}

func TestGitHubListCollaboratorsNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/nope/collaborators", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubCollaboratorService(srv)
	_, err := s.List(context.Background(), "octocat", "nope", forge.ListCollaboratorOpts{})
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

func TestGitHubAddCollaborator(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v3/repos/octocat/hello-world/collaborators/alice", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["permission"] != "push" {
			t.Errorf("expected permission push, got %s", req["permission"])
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubCollaboratorService(srv)
	err := s.Add(context.Background(), "octocat", "hello-world", "alice", forge.AddCollaboratorOpts{Permission: "push"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubRemoveCollaborator(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v3/repos/octocat/hello-world/collaborators/alice", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubCollaboratorService(srv)
	err := s.Remove(context.Background(), "octocat", "hello-world", "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubRemoveCollaboratorNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v3/repos/octocat/hello-world/collaborators/nobody", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubCollaboratorService(srv)
	err := s.Remove(context.Background(), "octocat", "hello-world", "nobody")
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

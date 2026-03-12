package gitlab

import (
	"context"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestGitLabListCollaborators(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/octocat%2Fhello-world/members", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*gitlab.ProjectMember{
			{ID: 1, Username: "alice", AccessLevel: gitlab.MaintainerPermissions},
			{ID: 2, Username: "bob", AccessLevel: gitlab.DeveloperPermissions},
			{ID: 3, Username: "carol", AccessLevel: gitlab.ReporterPermissions},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	collabs, err := f.Collaborators().List(context.Background(), "octocat", "hello-world", forge.ListCollaboratorOpts{})
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

func TestGitLabListCollaboratorsNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/octocat%2Fnope/members", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "404 Project Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Collaborators().List(context.Background(), "octocat", "nope", forge.ListCollaboratorOpts{})
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

func TestGitLabRemoveCollaborator(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/octocat%2Fhello-world/members", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*gitlab.ProjectMember{
			{ID: 42, Username: "alice", AccessLevel: gitlab.DeveloperPermissions},
		})
	})
	mux.HandleFunc("DELETE /api/v4/projects/octocat%2Fhello-world/members/42", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	err := f.Collaborators().Remove(context.Background(), "octocat", "hello-world", "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitLabRemoveCollaboratorNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/octocat%2Fhello-world/members", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*gitlab.ProjectMember{})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	err := f.Collaborators().Remove(context.Background(), "octocat", "hello-world", "nobody")
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

package gitea

import (
	"context"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.gitea.io/sdk/gitea"
)

func TestGiteaListCollaborators(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/octocat/hello-world/collaborators", func(w http.ResponseWriter, r *http.Request) {
		// Check if this is the permission check call (has a username suffix)
		_ = json.NewEncoder(w).Encode([]*gitea.User{
			{UserName: "alice"},
			{UserName: "bob"},
		})
	})
	mux.HandleFunc("GET /api/v1/repos/octocat/hello-world/collaborators/alice/permission", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(gitea.CollaboratorPermissionResult{Permission: "admin"})
	})
	mux.HandleFunc("GET /api/v1/repos/octocat/hello-world/collaborators/bob/permission", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(gitea.CollaboratorPermissionResult{Permission: "write"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	collabs, err := f.Collaborators().List(context.Background(), "octocat", "hello-world", forge.ListCollaboratorOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(collabs) != 2 {
		t.Fatalf("expected 2 collaborators, got %d", len(collabs))
	}
	assertEqual(t, "collabs[0].Login", "alice", collabs[0].Login)
	assertEqual(t, "collabs[0].Permission", "admin", collabs[0].Permission)
	assertEqual(t, "collabs[1].Login", "bob", collabs[1].Login)
	assertEqual(t, "collabs[1].Permission", "write", collabs[1].Permission)
}

func TestGiteaAddCollaborator(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("PUT /api/v1/repos/octocat/hello-world/collaborators/alice", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	err := f.Collaborators().Add(context.Background(), "octocat", "hello-world", "alice", forge.AddCollaboratorOpts{Permission: "write"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGiteaRemoveCollaborator(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("DELETE /api/v1/repos/octocat/hello-world/collaborators/alice", func(w http.ResponseWriter, r *http.Request) {
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

func TestGiteaRemoveCollaboratorNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("DELETE /api/v1/repos/octocat/hello-world/collaborators/nobody", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	err := f.Collaborators().Remove(context.Background(), "octocat", "hello-world", "nobody")
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

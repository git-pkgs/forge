package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v82/github"
)

func newTestGitHubFileService(srv *httptest.Server) *gitHubFileService {
	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	return &gitHubFileService{client: c}
}

func TestGitHubGetFile(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("hello world"))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/contents/README.md", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(github.RepositoryContent{
			Type:     ptr("file"),
			Name:     ptr("README.md"),
			Path:     ptr("README.md"),
			Content:  ptr(encoded),
			Encoding: ptr("base64"),
			SHA:      ptr("abc123"),
			Size:     ptrInt(11),
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubFileService(srv)
	file, err := s.Get(context.Background(), "octocat", "hello-world", "README.md", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "Name", "README.md", file.Name)
	assertEqual(t, "Path", "README.md", file.Path)
	assertEqual(t, "Content", "hello world", string(file.Content))
	assertEqual(t, "SHA", "abc123", file.SHA)
}

func TestGitHubGetFileNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/contents/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubFileService(srv)
	_, err := s.Get(context.Background(), "octocat", "hello-world", "nope", "")
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

func TestGitHubListFiles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/contents/src", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.RepositoryContent{
			{Type: ptr("file"), Name: ptr("main.go"), Path: ptr("src/main.go"), Size: ptrInt(1024)},
			{Type: ptr("dir"), Name: ptr("pkg"), Path: ptr("src/pkg"), Size: ptrInt(0)},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubFileService(srv)
	entries, err := s.List(context.Background(), "octocat", "hello-world", "src", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	assertEqual(t, "entries[0].Name", "main.go", entries[0].Name)
	assertEqual(t, "entries[0].Type", "file", entries[0].Type)
	assertEqual(t, "entries[1].Name", "pkg", entries[1].Name)
	assertEqual(t, "entries[1].Type", "dir", entries[1].Type)
}

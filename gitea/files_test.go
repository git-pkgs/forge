package gitea

import (
	"context"
	"encoding/base64"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGiteaGetFile(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("hello world"))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/contents/README.md", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":     "README.md",
			"path":     "README.md",
			"type":     "file",
			"encoding": "base64",
			"content":  encoded,
			"sha":      "abc123",
			"size":     11,
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	file, err := f.Files().Get(context.Background(), "testorg", "testrepo", "README.md", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "Name", "README.md", file.Name)
	assertEqual(t, "Content", "hello world", string(file.Content))
	assertEqual(t, "SHA", "abc123", file.SHA)
}

func TestGiteaGetFileNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/contents/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Files().Get(context.Background(), "testorg", "testrepo", "nope", "main")
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

func TestGiteaListFiles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/contents/src", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"name": "main.go", "path": "src/main.go", "type": "file", "size": 1024},
			{"name": "pkg", "path": "src/pkg", "type": "dir", "size": 0},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	entries, err := f.Files().List(context.Background(), "testorg", "testrepo", "src", "main")
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

package gitlab

import (
	"context"
	"encoding/base64"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitLabGetFile(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("hello world"))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo/repository/files/README.md", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"file_name": "README.md",
			"file_path": "README.md",
			"encoding":  "base64",
			"content":   encoded,
			"blob_id":   "abc123",
			"size":      11,
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	file, err := f.Files().Get(context.Background(), "mygroup", "myrepo", "README.md", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "Name", "README.md", file.Name)
	assertEqual(t, "Content", "hello world", string(file.Content))
	assertEqual(t, "SHA", "abc123", file.SHA)
}

func TestGitLabGetFileNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo/repository/files/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "404 File Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Files().Get(context.Background(), "mygroup", "myrepo", "nope", "main")
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

func TestGitLabListFiles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo/repository/tree", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"name": "main.go", "path": "src/main.go", "type": "blob"},
			{"name": "pkg", "path": "src/pkg", "type": "tree"},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	entries, err := f.Files().List(context.Background(), "mygroup", "myrepo", "src", "main")
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

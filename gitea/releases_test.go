package gitea

import (
	"context"
	"errors"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"code.gitea.io/sdk/gitea"
)

func TestCreateReleaseAlreadyExists(t *testing.T) {
	// Forgejo/Gitea return 409 {"message":"Release has no Tag"} when a
	// release already exists for the given tag. The message refers to an
	// internal is_tag flag and is meaningless to callers; surface the tag
	// name and the fact that it already exists instead.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("POST /api/v1/repos/o/r/releases", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"message":"Release has no Tag","url":"https://example.com/api/swagger"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "", nil)
	_, err := f.Releases().Create(context.Background(), "o", "r", forge.CreateReleaseOpts{
		TagName: "v1.0.0",
		Title:   "v1.0.0",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "v1.0.0") {
		t.Errorf("error should mention the tag name, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should say the release already exists, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "Release has no Tag") {
		t.Errorf("error should not surface the raw server message verbatim, got %q", err.Error())
	}
}

func TestCreateReleaseServerErrorBlankMessage(t *testing.T) {
	// Forgejo blanks the message on 500s in production mode, so the SDK
	// returns an error whose Error() is "". Callers must at least see the
	// HTTP status.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("POST /api/v1/repos/o/r/releases", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"","url":"https://example.com/api/swagger"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "", nil)
	_, err := f.Releases().Create(context.Background(), "o", "r", forge.CreateReleaseOpts{
		TagName: "v1.0.0",
		Title:   "v1.0.0",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() == "" {
		t.Fatalf("error string is empty")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should include the HTTP status, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "create release") {
		t.Errorf("error should name the operation, got %q", err.Error())
	}
}

func TestCreateReleaseServerErrorWithMessage(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("POST /api/v1/repos/o/r/releases", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"tag name is protected"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "", nil)
	_, err := f.Releases().Create(context.Background(), "o", "r", forge.CreateReleaseOpts{
		TagName: "v1.0.0",
		Title:   "v1.0.0",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "tag name is protected") {
		t.Errorf("error should keep the server message, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "422") {
		t.Errorf("error should include the HTTP status, got %q", err.Error())
	}
}

func TestCreateReleaseNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("POST /api/v1/repos/o/r/releases", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"repo not found"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "", nil)
	_, err := f.Releases().Create(context.Background(), "o", "r", forge.CreateReleaseOpts{
		TagName: "v1.0.0",
		Title:   "v1.0.0",
	})
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestConvertGiteaReleasePopulatesID(t *testing.T) {
	rel := convertGiteaRelease(&gitea.Release{
		ID:      777,
		TagName: "v7.7.7",
		Title:   "Release 7.7.7",
		IsDraft: true,
	})
	if rel.ID != 777 {
		t.Errorf("ID: want 777, got %d", rel.ID)
	}
	if rel.TagName != "v7.7.7" {
		t.Errorf("TagName: want %q, got %q", "v7.7.7", rel.TagName)
	}
}

func TestGiteaReleaseGetByIDFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/o/r/releases/tags/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"release not found"}`))
	})
	mux.HandleFunc("GET /api/v1/repos/o/r/releases/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":9,"tag_name":"v0.9.0-draft","name":"Draft 9","draft":true}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "", nil)
	rel, err := f.Releases().Get(context.Background(), "o", "r", "9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel.ID != 9 {
		t.Errorf("ID: want 9, got %d", rel.ID)
	}
	if rel.TagName != "v0.9.0-draft" {
		t.Errorf("TagName: want %q, got %q", "v0.9.0-draft", rel.TagName)
	}
	if !rel.Draft {
		t.Errorf("Draft: want true, got false")
	}
}

func TestGiteaReleaseUpdateByIDFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/o/r/releases/tags/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"release not found"}`))
	})
	mux.HandleFunc("GET /api/v1/repos/o/r/releases/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":9,"tag_name":"v0.9.0-draft","name":"Draft 9","draft":true}`))
	})
	mux.HandleFunc("PATCH /api/v1/repos/o/r/releases/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":9,"tag_name":"v0.9.0-draft","name":"Draft 9 renamed","draft":true}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "", nil)
	title := "Draft 9 renamed"
	rel, err := f.Releases().Update(context.Background(), "o", "r", "9", forge.UpdateReleaseOpts{
		Title: &title,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel.ID != 9 {
		t.Errorf("ID: want 9, got %d", rel.ID)
	}
	if rel.Title != "Draft 9 renamed" {
		t.Errorf("Title: want %q, got %q", "Draft 9 renamed", rel.Title)
	}
}

func TestGiteaReleaseDeleteByIDFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/o/r/releases/tags/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"release not found"}`))
	})
	mux.HandleFunc("GET /api/v1/repos/o/r/releases/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":9,"tag_name":"v0.9.0-draft","draft":true}`))
	})
	mux.HandleFunc("DELETE /api/v1/repos/o/r/releases/9", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "", nil)
	if err := f.Releases().Delete(context.Background(), "o", "r", "9"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGiteaReleaseUploadAssetByIDFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/o/r/releases/tags/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"release not found"}`))
	})
	mux.HandleFunc("GET /api/v1/repos/o/r/releases/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":9,"tag_name":"v0.9.0-draft","draft":true}`))
	})
	mux.HandleFunc("POST /api/v1/repos/o/r/releases/9/assets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":100,"name":"notes.txt","size":11,"browser_download_url":"https://example.com/o/r/releases/download/9/notes.txt"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(tmp, []byte("hello world"), 0o600); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	file, err := os.Open(tmp)
	if err != nil {
		t.Fatalf("opening temp file: %v", err)
	}
	defer func() { _ = file.Close() }()

	f := New(srv.URL, "", nil)
	asset, err := f.Releases().UploadAsset(context.Background(), "o", "r", "9", file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asset.Name != "notes.txt" {
		t.Errorf("Name: want %q, got %q", "notes.txt", asset.Name)
	}
	if asset.Size != 11 {
		t.Errorf("Size: want 11, got %d", asset.Size)
	}
}

func TestGiteaReleaseGetMissingTagNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/o/r/releases/tags/nope", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"release not found"}`))
	})
	mux.HandleFunc("GET /api/v1/repos/o/r/releases/tags/4242", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"release not found"}`))
	})
	mux.HandleFunc("GET /api/v1/repos/o/r/releases/4242", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"release not found"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "", nil)
	if _, err := f.Releases().Get(context.Background(), "o", "r", "nope"); !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("non-numeric missing tag: expected forge.ErrNotFound, got %v", err)
	}
	if _, err := f.Releases().Get(context.Background(), "o", "r", "4242"); !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("numeric missing tag: expected forge.ErrNotFound, got %v", err)
	}
}

package github

import (
	"context"
	"fmt"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-github/v82/github"
)

func TestGitHubReleaseGetByIDFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/tags/9", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message": "Not Found"}`)
	})
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"id": 9,
			"tag_name": "v0.9.0-draft",
			"name": "Draft 9",
			"body": "unpublished notes",
			"draft": true,
			"prerelease": false
		}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubReleaseService(srv)
	rel, err := svc.Get(context.Background(), "octocat", "hello", "9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "TagName", "v0.9.0-draft", rel.TagName)
	assertEqualBool(t, "Draft", true, rel.Draft)
	assertEqualInt64(t, "ID", 9, rel.ID)
}

func TestGitHubReleaseUploadAssetByIDFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/tags/9", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message": "Not Found"}`)
	})
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id": 9, "tag_name": "v0.9.0-draft", "draft": true}`)
	})
	// go-github posts uploads to <base>/api/uploads/... on enterprise URLs, so the stub must match that path.
	mux.HandleFunc("POST /api/v3/api/uploads/repos/octocat/hello/releases/9/assets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{
			"id": 100,
			"name": "notes.txt",
			"size": 11,
			"download_count": 0,
			"browser_download_url": "https://github.com/octocat/hello/releases/download/9/notes.txt"
		}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(tmp, []byte("hello world"), 0o600); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	f, err := os.Open(tmp)
	if err != nil {
		t.Fatalf("opening temp file: %v", err)
	}
	defer func() { _ = f.Close() }()

	svc := newTestGitHubReleaseService(srv)
	asset, err := svc.UploadAsset(context.Background(), "octocat", "hello", "9", f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "Asset.Name", "notes.txt", asset.Name)
	assertEqualInt(t, "Asset.Size", 11, asset.Size)
}

func TestGitHubReleaseUpdateByIDFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/tags/9", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message": "Not Found"}`)
	})
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id": 9, "tag_name": "v0.9.0-draft", "draft": true}`)
	})
	mux.HandleFunc("PATCH /api/v3/repos/octocat/hello/releases/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id": 9, "tag_name": "v0.9.0-draft", "name": "Draft 9 renamed", "draft": true}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubReleaseService(srv)
	rel, err := svc.Update(context.Background(), "octocat", "hello", "9", forge.UpdateReleaseOpts{
		Title: ptr("Draft 9 renamed"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualInt64(t, "ID", 9, rel.ID)
	assertEqual(t, "Title", "Draft 9 renamed", rel.Title)
}

func TestGitHubReleaseDeleteByIDFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/tags/9", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message": "Not Found"}`)
	})
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id": 9, "tag_name": "v0.9.0-draft", "draft": true}`)
	})
	mux.HandleFunc("DELETE /api/v3/repos/octocat/hello/releases/9", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubReleaseService(srv)
	if err := svc.Delete(context.Background(), "octocat", "hello", "9"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubReleaseGetMissingTagNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/tags/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message": "Not Found"}`)
	})
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/tags/4242", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message": "Not Found"}`)
	})
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/4242", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"message": "Not Found"}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubReleaseService(srv)
	if _, err := svc.Get(context.Background(), "octocat", "hello", "nope"); err != forge.ErrNotFound {
		t.Errorf("non-numeric missing tag: expected forge.ErrNotFound, got %v", err)
	}
	if _, err := svc.Get(context.Background(), "octocat", "hello", "4242"); err != forge.ErrNotFound {
		t.Errorf("numeric missing tag: expected forge.ErrNotFound, got %v", err)
	}
}

func TestConvertGitHubReleasePopulatesID(t *testing.T) {
	rel := convertGitHubRelease(&github.RepositoryRelease{
		ID:      ptrInt64(12345),
		TagName: ptr("v9.9.9"),
		Name:    ptr("Draft 9"),
		Draft:   ptrBool(true),
	})
	assertEqualInt64(t, "ID", 12345, rel.ID)
	assertEqual(t, "TagName", "v9.9.9", rel.TagName)
	assertEqualBool(t, "Draft", true, rel.Draft)
}

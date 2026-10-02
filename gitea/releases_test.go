package gitea

import (
	"context"
	"errors"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

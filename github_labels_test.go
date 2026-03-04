package forges

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v82/github"
)

func newTestGitHubLabelService(srv *httptest.Server) *gitHubLabelService {
	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	return &gitHubLabelService{client: c}
}

func TestGitHubListLabels(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/labels", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.Label{
			{Name: ptr("bug"), Color: ptr("d73a4a"), Description: ptr("Something isn't working")},
			{Name: ptr("enhancement"), Color: ptr("a2eeef"), Description: ptr("New feature")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubLabelService(srv)
	labels, err := s.List(context.Background(), "octocat", "hello-world", ListLabelOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(labels))
	}
	assertEqual(t, "labels[0].Name", "bug", labels[0].Name)
	assertEqual(t, "labels[0].Color", "d73a4a", labels[0].Color)
	assertEqual(t, "labels[1].Name", "enhancement", labels[1].Name)
}

func TestGitHubGetLabel(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/labels/bug", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(github.Label{
			Name:        ptr("bug"),
			Color:       ptr("d73a4a"),
			Description: ptr("Something isn't working"),
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubLabelService(srv)
	label, err := s.Get(context.Background(), "octocat", "hello-world", "bug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "Name", "bug", label.Name)
	assertEqual(t, "Color", "d73a4a", label.Color)
	assertEqual(t, "Description", "Something isn't working", label.Description)
}

func TestGitHubGetLabelNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/labels/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubLabelService(srv)
	_, err := s.Get(context.Background(), "octocat", "hello-world", "nope")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGitHubCreateLabel(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello-world/labels", func(w http.ResponseWriter, r *http.Request) {
		var req github.Label
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(github.Label{
			Name:        req.Name,
			Color:       req.Color,
			Description: req.Description,
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubLabelService(srv)
	label, err := s.Create(context.Background(), "octocat", "hello-world", CreateLabelOpts{
		Name:  "priority",
		Color: "ff0000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "Name", "priority", label.Name)
	assertEqual(t, "Color", "ff0000", label.Color)
}

func TestGitHubUpdateLabel(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v3/repos/octocat/hello-world/labels/bug", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(github.Label{
			Name:  ptr("bug"),
			Color: ptr("00ff00"),
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubLabelService(srv)
	newColor := "00ff00"
	label, err := s.Update(context.Background(), "octocat", "hello-world", "bug", UpdateLabelOpts{
		Color: &newColor,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "Color", "00ff00", label.Color)
}

func TestGitHubDeleteLabel(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v3/repos/octocat/hello-world/labels/bug", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubLabelService(srv)
	if err := s.Delete(context.Background(), "octocat", "hello-world", "bug"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

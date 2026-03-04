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

func newTestGitHubMilestoneService(srv *httptest.Server) *gitHubMilestoneService {
	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	return &gitHubMilestoneService{client: c}
}

func TestGitHubListMilestones(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/milestones", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.Milestone{
			{Number: ptrInt(1), Title: ptr("v1.0"), State: ptr("open"), Description: ptr("First release")},
			{Number: ptrInt(2), Title: ptr("v2.0"), State: ptr("open"), Description: ptr("Second release")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubMilestoneService(srv)
	milestones, err := s.List(context.Background(), "octocat", "hello-world", forge.ListMilestoneOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(milestones) != 2 {
		t.Fatalf("expected 2 milestones, got %d", len(milestones))
	}
	assertEqual(t, "milestones[0].Title", "v1.0", milestones[0].Title)
	assertEqualInt(t, "milestones[0].Number", 1, milestones[0].Number)
	assertEqual(t, "milestones[1].Title", "v2.0", milestones[1].Title)
}

func TestGitHubGetMilestone(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/milestones/1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(github.Milestone{
			Number:      ptrInt(1),
			Title:       ptr("v1.0"),
			State:       ptr("open"),
			Description: ptr("First release"),
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubMilestoneService(srv)
	milestone, err := s.Get(context.Background(), "octocat", "hello-world", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "Title", "v1.0", milestone.Title)
	assertEqualInt(t, "Number", 1, milestone.Number)
	assertEqual(t, "State", "open", milestone.State)
	assertEqual(t, "Description", "First release", milestone.Description)
}

func TestGitHubGetMilestoneNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/milestones/999", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubMilestoneService(srv)
	_, err := s.Get(context.Background(), "octocat", "hello-world", 999)
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

func TestGitHubCreateMilestone(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello-world/milestones", func(w http.ResponseWriter, r *http.Request) {
		var req github.Milestone
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(github.Milestone{
			Number:      ptrInt(3),
			Title:       req.Title,
			Description: req.Description,
			State:       ptr("open"),
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubMilestoneService(srv)
	milestone, err := s.Create(context.Background(), "octocat", "hello-world", forge.CreateMilestoneOpts{
		Title:       "v3.0",
		Description: "Third release",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualInt(t, "Number", 3, milestone.Number)
	assertEqual(t, "Title", "v3.0", milestone.Title)
	assertEqual(t, "State", "open", milestone.State)
}

func TestGitHubCloseMilestone(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v3/repos/octocat/hello-world/milestones/1", func(w http.ResponseWriter, r *http.Request) {
		var req github.Milestone
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.GetState() != "closed" {
			t.Errorf("expected state=closed, got %s", req.GetState())
		}
		_ = json.NewEncoder(w).Encode(github.Milestone{
			Number: ptrInt(1),
			State:  ptr("closed"),
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubMilestoneService(srv)
	if err := s.Close(context.Background(), "octocat", "hello-world", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubDeleteMilestone(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v3/repos/octocat/hello-world/milestones/1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubMilestoneService(srv)
	if err := s.Delete(context.Background(), "octocat", "hello-world", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

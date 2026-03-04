package github

import (
	"context"
	"fmt"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v82/github"
)

func newTestGitHubCIService(srv *httptest.Server) *gitHubCIService {
	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	return &gitHubCIService{client: c}
}

func TestGitHubCIListRuns(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/actions/runs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"total_count": 1,
			"workflow_runs": [
				{
					"id": 123,
					"name": "CI",
					"status": "completed",
					"conclusion": "success",
					"head_branch": "main",
					"head_sha": "abc123",
					"event": "push",
					"html_url": "https://github.com/octocat/hello/actions/runs/123",
					"actor": {"login": "octocat"}
				}
			]
		}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubCIService(srv)
	runs, err := svc.ListRuns(context.Background(), "octocat", "hello", forge.ListCIRunOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	r := runs[0]
	if r.ID != 123 {
		t.Errorf("ID: want 123, got %d", r.ID)
	}
	assertEqual(t, "Title", "CI", r.Title)
	assertEqual(t, "Status", "completed", r.Status)
	assertEqual(t, "Conclusion", "success", r.Conclusion)
	assertEqual(t, "Branch", "main", r.Branch)
	assertEqual(t, "Event", "push", r.Event)
	assertEqual(t, "Author.Login", "octocat", r.Author.Login)
}

func TestGitHubCIGetRun(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/actions/runs/123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"id": 123,
			"name": "CI",
			"status": "completed",
			"conclusion": "success",
			"head_branch": "main",
			"head_sha": "abc123"
		}`)
	})
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/actions/runs/123/jobs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"total_count": 1,
			"jobs": [
				{
					"id": 456,
					"name": "build",
					"status": "completed",
					"conclusion": "success"
				}
			]
		}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubCIService(srv)
	run, err := svc.GetRun(context.Background(), "octocat", "hello", 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.ID != 123 {
		t.Errorf("ID: want 123, got %d", run.ID)
	}
	if len(run.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(run.Jobs))
	}
	assertEqual(t, "Jobs[0].Name", "build", run.Jobs[0].Name)
}

func TestGitHubCICancelRun(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello/actions/runs/123/cancel", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprint(w, `{}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubCIService(srv)
	err := svc.CancelRun(context.Background(), "octocat", "hello", 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

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

func TestGitHubListCommitStatuses(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/commits/abc123/statuses", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.RepoStatus{
			{
				State:       ptr("success"),
				Context:     ptr("ci/tests"),
				Description: ptr("All tests passed"),
				TargetURL:   ptr("https://ci.example.com/1"),
				Creator:     &github.User{Login: ptr("bot")},
			},
			{
				State:       ptr("pending"),
				Context:     ptr("ci/lint"),
				Description: ptr("Running"),
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	s := &gitHubCommitStatusService{client: c}

	statuses, err := s.List(context.Background(), "octocat", "hello-world", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}
	assertEqual(t, "statuses[0].State", "success", statuses[0].State)
	assertEqual(t, "statuses[0].Context", "ci/tests", statuses[0].Context)
	assertEqual(t, "statuses[0].Description", "All tests passed", statuses[0].Description)
	assertEqual(t, "statuses[0].TargetURL", "https://ci.example.com/1", statuses[0].TargetURL)
	assertEqual(t, "statuses[0].Creator", "bot", statuses[0].Creator)
	assertEqual(t, "statuses[1].State", "pending", statuses[1].State)
	assertEqual(t, "statuses[1].Context", "ci/lint", statuses[1].Context)
}

func TestGitHubSetCommitStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello-world/statuses/abc123", func(w http.ResponseWriter, r *http.Request) {
		var body github.RepoStatus
		_ = json.NewDecoder(r.Body).Decode(&body)
		body.Creator = &github.User{Login: ptr("bot")}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(&body)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	s := &gitHubCommitStatusService{client: c}

	status, err := s.Set(context.Background(), "octocat", "hello-world", "abc123", forge.SetCommitStatusOpts{
		State:       "success",
		Context:     "my-check",
		Description: "Everything passed",
		TargetURL:   "https://example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "State", "success", status.State)
	assertEqual(t, "Context", "my-check", status.Context)
	assertEqual(t, "Description", "Everything passed", status.Description)
	assertEqual(t, "Creator", "bot", status.Creator)
}

func TestGitHubListCommitStatusesNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/nope/commits/abc123/statuses", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	s := &gitHubCommitStatusService{client: c}

	_, err := s.List(context.Background(), "octocat", "nope", "abc123")
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

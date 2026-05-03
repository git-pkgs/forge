package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGiteaUpdateIssueReplacesLabels(t *testing.T) {
	var replaceCalled bool
	var replaceBody struct {
		Labels []int64 `json:"labels"`
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/octocat/hello-world/labels", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": 7, "name": "ready-for-agent", "color": "#0e8a16"},
			{"id": 8, "name": "needs-triage", "color": "#ededed"},
		})
	})
	mux.HandleFunc("PUT /api/v1/repos/octocat/hello-world/issues/42/labels", func(w http.ResponseWriter, r *http.Request) {
		replaceCalled = true
		_ = json.NewDecoder(r.Body).Decode(&replaceBody)
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": 7, "name": "ready-for-agent", "color": "#0e8a16"},
		})
	})
	mux.HandleFunc("GET /api/v1/repos/octocat/hello-world/issues/42", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"number": 42,
			"title":  "the issue",
			"state":  "open",
			"labels": []map[string]any{
				{"id": 7, "name": "ready-for-agent", "color": "#0e8a16"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	issue, err := f.Issues().Update(context.Background(), "octocat", "hello-world", 42, forge.UpdateIssueOpts{
		Labels: []string{"ready-for-agent"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !replaceCalled {
		t.Fatal("expected ReplaceIssueLabels (PUT /issues/42/labels) to be called")
	}
	if len(replaceBody.Labels) != 1 || replaceBody.Labels[0] != 7 {
		t.Errorf("expected PUT body labels=[7], got %v", replaceBody.Labels)
	}
	if len(issue.Labels) != 1 || issue.Labels[0].Name != "ready-for-agent" {
		t.Errorf("expected returned issue to have label ready-for-agent, got %+v", issue.Labels)
	}
}

func TestGiteaCreateIssueWithMilestone(t *testing.T) {
	var createBody struct {
		Title     string `json:"title"`
		Milestone int64  `json:"milestone"`
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/octocat/hello-world/milestones/v1.0", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 3, "title": "v1.0", "state": "open"})
	})
	mux.HandleFunc("POST /api/v1/repos/octocat/hello-world/issues", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&createBody)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"number":    42,
			"title":     createBody.Title,
			"state":     "open",
			"milestone": map[string]any{"id": 3, "title": "v1.0", "state": "open"},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	issue, err := f.Issues().Create(context.Background(), "octocat", "hello-world", forge.CreateIssueOpts{
		Title:     "the issue",
		Milestone: "v1.0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createBody.Milestone != 3 {
		t.Errorf("expected POST body milestone=3, got %d", createBody.Milestone)
	}
	if issue.Milestone == nil || issue.Milestone.Title != "v1.0" {
		t.Errorf("expected returned issue to have milestone v1.0, got %+v", issue.Milestone)
	}
}

func TestGiteaUpdateIssueSetsMilestone(t *testing.T) {
	var editCalled bool
	var editBody struct {
		Milestone *int64 `json:"milestone"`
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/octocat/hello-world/milestones/v1.0", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 3, "title": "v1.0", "state": "open"})
	})
	mux.HandleFunc("PATCH /api/v1/repos/octocat/hello-world/issues/42", func(w http.ResponseWriter, r *http.Request) {
		editCalled = true
		_ = json.NewDecoder(r.Body).Decode(&editBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"number":    42,
			"title":     "the issue",
			"state":     "open",
			"milestone": map[string]any{"id": 3, "title": "v1.0", "state": "open"},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	ms := "v1.0"
	issue, err := f.Issues().Update(context.Background(), "octocat", "hello-world", 42, forge.UpdateIssueOpts{
		Milestone: &ms,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !editCalled {
		t.Fatal("expected EditIssue (PATCH /issues/42) to be called")
	}
	if editBody.Milestone == nil || *editBody.Milestone != 3 {
		t.Errorf("expected PATCH body milestone=3, got %v", editBody.Milestone)
	}
	if issue.Milestone == nil || issue.Milestone.Title != "v1.0" {
		t.Errorf("expected returned issue to have milestone v1.0, got %+v", issue.Milestone)
	}
}

func TestGiteaUpdateIssueClearsMilestone(t *testing.T) {
	var editBody struct {
		Milestone *int64 `json:"milestone"`
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("PATCH /api/v1/repos/octocat/hello-world/issues/42", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&editBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"number": 42, "title": "the issue", "state": "open"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	ms := ""
	_, err := f.Issues().Update(context.Background(), "octocat", "hello-world", 42, forge.UpdateIssueOpts{
		Milestone: &ms,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if editBody.Milestone == nil || *editBody.Milestone != 0 {
		t.Errorf("expected PATCH body milestone=0 (clear), got %v", editBody.Milestone)
	}
}

func TestGiteaUpdateIssueUnknownMilestone(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/octocat/hello-world/milestones/nope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	ms := "nope"
	_, err := f.Issues().Update(context.Background(), "octocat", "hello-world", 42, forge.UpdateIssueOpts{
		Milestone: &ms,
	})
	if err == nil {
		t.Fatal("expected error for unknown milestone")
	}
}

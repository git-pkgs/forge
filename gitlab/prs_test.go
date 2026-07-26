package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGitLabCreatePRWithAssigneesAndReviewers(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/users", func(w http.ResponseWriter, r *http.Request) {
		uname := r.URL.Query().Get("username")
		switch uname {
		case "alice":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 101, "username": "alice"}})
		case "bob":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 202, "username": "bob"}})
		default:
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		}
	})

	var gotAssigneeIDs, gotReviewerIDs []int64
	mux.HandleFunc("POST /api/v4/projects/mygroup%2Fmyrepo/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Title        string  `json:"title"`
			AssigneeIDs  []int64 `json:"assignee_ids"`
			ReviewerIDs  []int64 `json:"reviewer_ids"`
			SourceBranch string  `json:"source_branch"`
			TargetBranch string  `json:"target_branch"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotAssigneeIDs = req.AssigneeIDs
		gotReviewerIDs = req.ReviewerIDs
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":            1,
			"iid":           1,
			"title":         req.Title,
			"state":         "opened",
			"source_branch": req.SourceBranch,
			"target_branch": req.TargetBranch,
			"assignees": []map[string]any{
				{"id": 101, "username": "alice"},
			},
			"reviewers": []map[string]any{
				{"id": 202, "username": "bob"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)

	pr, err := f.PullRequests().Create(context.Background(), "mygroup", "myrepo", forge.CreatePROpts{
		Title:     "New Feature",
		Head:      "feature",
		Base:      "main",
		Assignees: []string{"alice"},
		Reviewers: []string{"bob"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotAssigneeIDs) != 1 || gotAssigneeIDs[0] != 101 {
		t.Fatalf("expected assignee_ids [101], got %v", gotAssigneeIDs)
	}
	if len(gotReviewerIDs) != 1 || gotReviewerIDs[0] != 202 {
		t.Fatalf("expected reviewer_ids [202], got %v", gotReviewerIDs)
	}
	if len(pr.Assignees) != 1 || pr.Assignees[0].Login != "alice" {
		t.Fatalf("expected 1 assignee alice, got %v", pr.Assignees)
	}
	if len(pr.Reviewers) != 1 || pr.Reviewers[0].Login != "bob" {
		t.Fatalf("expected 1 reviewer bob, got %v", pr.Reviewers)
	}
}

func TestGitLabUpdatePRWithAssigneesAndReviewers(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/users", func(w http.ResponseWriter, r *http.Request) {
		uname := r.URL.Query().Get("username")
		switch uname {
		case "alice":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 101, "username": "alice"}})
		case "bob":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 202, "username": "bob"}})
		default:
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		}
	})

	var gotAssigneeIDs, gotReviewerIDs []int64
	mux.HandleFunc("PUT /api/v4/projects/mygroup%2Fmyrepo/merge_requests/1", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			AssigneeIDs []int64 `json:"assignee_ids"`
			ReviewerIDs []int64 `json:"reviewer_ids"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotAssigneeIDs = req.AssigneeIDs
		gotReviewerIDs = req.ReviewerIDs
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    1,
			"iid":   1,
			"title": "Updated MR",
			"state": "opened",
			"assignees": []map[string]any{
				{"id": 101, "username": "alice"},
			},
			"reviewers": []map[string]any{
				{"id": 202, "username": "bob"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)

	pr, err := f.PullRequests().Update(context.Background(), "mygroup", "myrepo", 1, forge.UpdatePROpts{
		Assignees: []string{"alice"},
		Reviewers: []string{"bob"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotAssigneeIDs) != 1 || gotAssigneeIDs[0] != 101 {
		t.Fatalf("expected assignee_ids [101], got %v", gotAssigneeIDs)
	}
	if len(gotReviewerIDs) != 1 || gotReviewerIDs[0] != 202 {
		t.Fatalf("expected reviewer_ids [202], got %v", gotReviewerIDs)
	}
	if len(pr.Assignees) != 1 || pr.Assignees[0].Login != "alice" {
		t.Fatalf("expected 1 assignee alice, got %v", pr.Assignees)
	}
}

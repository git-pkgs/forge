package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGitLabCreateIssueWithAssignees(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/users", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("username") == "alice" {
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 42, "username": "alice"}})
		} else {
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		}
	})
	var gotAssigneeIDs []int64
	mux.HandleFunc("POST /api/v4/projects/mygroup%2Fmyrepo/issues", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Title       string  `json:"title"`
			AssigneeIDs []int64 `json:"assignee_ids"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotAssigneeIDs = req.AssigneeIDs
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    1,
			"iid":   1,
			"title": req.Title,
			"state": "opened",
			"assignees": []map[string]any{
				{"id": 42, "username": "alice"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)

	iss, err := f.Issues().Create(context.Background(), "mygroup", "myrepo", forge.CreateIssueOpts{
		Title:     "Test issue",
		Body:      "Body text",
		Assignees: []string{"alice"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotAssigneeIDs) != 1 || gotAssigneeIDs[0] != 42 {
		t.Fatalf("expected assignee_ids [42], got %v", gotAssigneeIDs)
	}
	if len(iss.Assignees) != 1 || iss.Assignees[0].Login != "alice" {
		t.Fatalf("expected 1 assignee alice, got %v", iss.Assignees)
	}
}

func TestGitLabListIssuesWithOpenState(t *testing.T) {
	var gotState string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo/issues", func(w http.ResponseWriter, r *http.Request) {
		gotState = r.URL.Query().Get("state")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":               1,
				"iid":              1,
				"title":            "Test issue",
				"description":      "Body",
				"state":            "opened",
				"web_url":          "https://gitlab.com/mygroup/myrepo/-/issues/1",
				"updated_at":       "2024-01-01T00:00:00Z",
				"user_notes_count": 0,
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)

	issues, err := f.Issues().List(context.Background(), "mygroup", "myrepo", forge.ListIssueOpts{
		State: "open",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	assertEqual(t, "state query param", "opened", gotState)
	assertEqual(t, "issue state", "open", issues[0].State)
}

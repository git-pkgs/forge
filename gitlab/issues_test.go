package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGitLabCreateIssueRejectsAssignees(t *testing.T) {
	srv := httptest.NewServer(nil)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)

	_, err := f.Issues().Create(context.Background(), "mygroup", "myrepo", forge.CreateIssueOpts{
		Title:     "Test issue",
		Body:      "Body text",
		Assignees: []string{"someuser"},
	})
	if err == nil {
		t.Fatal("expected error when assignees are set, got nil")
	}
	if !strings.Contains(err.Error(), "assignee IDs") {
		t.Fatalf("expected error to mention 'assignee IDs', got: %v", err)
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

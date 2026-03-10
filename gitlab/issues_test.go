package gitlab

import (
	"context"
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

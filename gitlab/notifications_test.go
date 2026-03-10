package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGitLabListNotifications(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/todos", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":          1,
				"body":        "You were mentioned in issue #5",
				"action_name": "mentioned",
				"target_type": "Issue",
				"target_url":  "https://gitlab.com/mygroup/myrepo/-/issues/5",
				"target":      map[string]any{"id": 5},
				"state":       "pending",
				"project": map[string]any{
					"id":                  10,
					"name":                "myrepo",
					"path_with_namespace": "mygroup/myrepo",
				},
				"created_at": "2024-06-01T10:00:00Z",
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	notifications, err := f.Notifications().List(context.Background(), forge.ListNotificationOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifications))
	}

	assertEqual(t, "ID", "1", notifications[0].ID)
	assertEqual(t, "Title", "You were mentioned in issue #5", notifications[0].Title)
	assertEqual(t, "Reason", "mentioned", notifications[0].Reason)
	assertEqual(t, "SubjectType", "issue", string(notifications[0].SubjectType))
	assertEqual(t, "Repo", "mygroup/myrepo", notifications[0].Repo)
	assertEqualBool(t, "Unread", true, notifications[0].Unread)
}

func TestGitLabListNotificationsUnread(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/todos", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != "pending" {
			t.Errorf("expected state=pending, got %q", r.URL.Query().Get("state"))
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Notifications().List(context.Background(), forge.ListNotificationOpts{Unread: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitLabMarkNotificationRead(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v4/todos/42/mark_as_done", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	err := f.Notifications().MarkRead(context.Background(), forge.MarkNotificationOpts{ID: "42"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitLabMarkAllNotificationsRead(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v4/todos/mark_as_done", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	err := f.Notifications().MarkRead(context.Background(), forge.MarkNotificationOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConvertGitLabTodoTargetType(t *testing.T) {
	tests := []struct {
		input string
		want  forge.NotificationSubjectType
	}{
		{"Issue", forge.NotificationSubjectIssue},
		{"MergeRequest", forge.NotificationSubjectPullRequest},
		{"Commit", forge.NotificationSubjectCommit},
	}

	for _, tt := range tests {
		got := convertGitLabTodoTargetType(tt.input)
		if got != tt.want {
			t.Errorf("convertGitLabTodoTargetType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGiteaListNotifications(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/notifications", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":     1,
				"unread": true,
				"subject": map[string]any{
					"title":    "Bug fix",
					"type":     "Issue",
					"html_url": "https://codeberg.org/testorg/testrepo/issues/1",
				},
				"repository": map[string]any{
					"full_name": "testorg/testrepo",
					"name":      "testrepo",
					"owner":     map[string]any{"login": "testorg"},
				},
				"updated_at": "2024-06-01T10:00:00Z",
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
	assertEqualBool(t, "Unread", true, notifications[0].Unread)
	assertEqual(t, "Title", "Bug fix", notifications[0].Title)
	assertEqual(t, "SubjectType", "issue", string(notifications[0].SubjectType))
	assertEqual(t, "Repo", "testorg/testrepo", notifications[0].Repo)
}

func TestGiteaListNotificationsForRepo(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/notifications", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":     2,
				"unread": false,
				"subject": map[string]any{
					"title":    "New PR",
					"type":     "Pull",
					"html_url": "https://codeberg.org/testorg/testrepo/pulls/5",
				},
				"repository": map[string]any{
					"full_name": "testorg/testrepo",
					"name":      "testrepo",
					"owner":     map[string]any{"login": "testorg"},
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	notifications, err := f.Notifications().List(context.Background(), forge.ListNotificationOpts{
		Repo: "testorg/testrepo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifications))
	}
	assertEqual(t, "SubjectType", "pull_request", string(notifications[0].SubjectType))
}

func TestGiteaGetNotification(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/notifications/threads/42", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     42,
			"unread": true,
			"subject": map[string]any{
				"title": "Update docs",
				"type":  "Issue",
			},
			"repository": map[string]any{
				"full_name": "testorg/testrepo",
				"name":      "testrepo",
				"owner":     map[string]any{"login": "testorg"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	n, err := f.Notifications().Get(context.Background(), "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "ID", "42", n.ID)
	assertEqual(t, "Title", "Update docs", n.Title)
	assertEqualBool(t, "Unread", true, n.Unread)
}

func TestGiteaGetNotificationNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/notifications/threads/999", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Notifications().Get(context.Background(), "999")
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

func TestGiteaMarkNotificationRead(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("PATCH /api/v1/notifications/threads/42", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     42,
			"unread": false,
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	err := f.Notifications().MarkRead(context.Background(), forge.MarkNotificationOpts{ID: "42"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGiteaMarkAllNotificationsRead(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("PUT /api/v1/notifications", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	err := f.Notifications().MarkRead(context.Background(), forge.MarkNotificationOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConvertGiteaSubjectType(t *testing.T) {
	tests := []struct {
		input string
		want  forge.NotificationSubjectType
	}{
		{"Issue", forge.NotificationSubjectIssue},
		{"Pull", forge.NotificationSubjectPullRequest},
		{"Commit", forge.NotificationSubjectCommit},
		{"Repository", forge.NotificationSubjectRepository},
	}

	for _, tt := range tests {
		got := convertGiteaSubjectType(tt.input)
		if got != tt.want {
			t.Errorf("convertGiteaSubjectType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

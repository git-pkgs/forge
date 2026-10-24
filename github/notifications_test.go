package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
	"github.com/google/go-github/v82/github"
)

func newTestGitHubNotificationService(srv *httptest.Server) *gitHubNotificationService {
	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	return &gitHubNotificationService{client: c}
}

func TestGitHubListNotifications(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/notifications", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.Notification{
			{
				ID:     ptr("1"),
				Unread: ptrBool(true),
				Reason: ptr("mention"),
				Subject: &github.NotificationSubject{
					Title: ptr("Fix bug"),
					Type:  ptr("Issue"),
					URL:   ptr("https://api.github.com/repos/octocat/hello-world/issues/123"),
				},
				Repository: &github.Repository{
					FullName: ptr("octocat/hello-world"),
					HTMLURL:  ptr("https://github.com/octocat/hello-world"),
				},
				UpdatedAt: &github.Timestamp{Time: parseTime("2024-06-01T10:00:00Z")},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubNotificationService(srv)
	notifications, err := s.List(context.Background(), forge.ListNotificationOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifications))
	}

	assertEqual(t, "ID", "1", notifications[0].ID)
	assertEqualBool(t, "Unread", true, notifications[0].Unread)
	assertEqual(t, "Reason", "mention", notifications[0].Reason)
	assertEqual(t, "Title", "Fix bug", notifications[0].Title)
	assertEqual(t, "SubjectType", "issue", string(notifications[0].SubjectType))
	assertEqualInt(t, "SubjectNumber", 123, notifications[0].SubjectNumber)
	assertEqual(t, "URL", "https://github.com/octocat/hello-world/issues/123", notifications[0].URL)
	assertEqual(t, "Repo", "octocat/hello-world", notifications[0].Repo)
	assertEqual(t, "RepoURL", "https://github.com/octocat/hello-world", notifications[0].RepoURL)
}

func TestGitHubSubjectLink(t *testing.T) {
	const repo = "https://github.com/octocat/hello"
	tests := []struct {
		name       string
		repoURL    string
		apiURL     string
		wantURL    string
		wantNumber int
	}{
		{"issue", repo, "https://api.github.com/repos/octocat/hello/issues/12", repo + "/issues/12", 12},
		{"pull request", repo, "https://api.github.com/repos/octocat/hello/pulls/42", repo + "/pull/42", 42},
		{"discussion", repo, "https://api.github.com/repos/octocat/hello/discussions/7", repo + "/discussions/7", 7},
		{"commit", repo, "https://api.github.com/repos/octocat/hello/commits/abc123", repo + "/commit/abc123", 0},
		{"release by ID", repo, "https://api.github.com/repos/octocat/hello/releases/987654", "", 0},
		{"enterprise", "https://ghe.example.com/octocat/hello", "https://ghe.example.com/api/v3/repos/octocat/hello/pulls/5",
			"https://ghe.example.com/octocat/hello/pull/5", 5},
		{"trailing slash on repo URL", repo + "/", "https://api.github.com/repos/octocat/hello/issues/3", repo + "/issues/3", 3},
		{"no subject URL", repo, "", "", 0},
		{"no repository URL", "", "https://api.github.com/repos/octocat/hello/issues/12", "", 0},
		{"non-numeric number", repo, "https://api.github.com/repos/octocat/hello/issues/abc", "", 0},
		{"not a repository path", repo, "https://api.github.com/notifications/threads/1", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotNumber := gitHubSubjectLink(tt.repoURL, tt.apiURL)
			assertEqual(t, "URL", tt.wantURL, gotURL)
			assertEqualInt(t, "Number", tt.wantNumber, gotNumber)
		})
	}
}

func TestGitHubListNotificationsUnread(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("all") == "true" {
			t.Error("expected all=false for unread filter")
		}
		_ = json.NewEncoder(w).Encode([]*github.Notification{})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubNotificationService(srv)
	_, err := s.List(context.Background(), forge.ListNotificationOpts{Unread: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubListNotificationsForRepo(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/notifications", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.Notification{
			{
				ID:     ptr("2"),
				Unread: ptrBool(false),
				Subject: &github.NotificationSubject{
					Title: ptr("New release"),
					Type:  ptr("Release"),
				},
				Repository: &github.Repository{
					FullName: ptr("octocat/hello-world"),
					HTMLURL:  ptr("https://github.com/octocat/hello-world"),
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubNotificationService(srv)
	notifications, err := s.List(context.Background(), forge.ListNotificationOpts{Repo: "octocat/hello-world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifications))
	}
	assertEqual(t, "SubjectType", "release", string(notifications[0].SubjectType))
}

func TestGitHubGetNotification(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/notifications/threads/42", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(&github.Notification{
			ID:     ptr("42"),
			Unread: ptrBool(true),
			Reason: ptr("assign"),
			Subject: &github.NotificationSubject{
				Title: ptr("Add feature"),
				Type:  ptr("PullRequest"),
			},
			Repository: &github.Repository{
				FullName: ptr("octocat/hello-world"),
				HTMLURL:  ptr("https://github.com/octocat/hello-world"),
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubNotificationService(srv)
	n, err := s.Get(context.Background(), "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "ID", "42", n.ID)
	assertEqual(t, "Title", "Add feature", n.Title)
	assertEqual(t, "SubjectType", "pull_request", string(n.SubjectType))
}

func TestGitHubGetNotificationNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/notifications/threads/999", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubNotificationService(srv)
	_, err := s.Get(context.Background(), "999")
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

func TestGitHubMarkNotificationRead(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v3/notifications/threads/42", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusResetContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubNotificationService(srv)
	err := s.MarkRead(context.Background(), forge.MarkNotificationOpts{ID: "42"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubMarkAllNotificationsRead(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v3/notifications", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusResetContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubNotificationService(srv)
	err := s.MarkRead(context.Background(), forge.MarkNotificationOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubMarkRepoNotificationsRead(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v3/repos/octocat/hello-world/notifications", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusResetContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubNotificationService(srv)
	err := s.MarkRead(context.Background(), forge.MarkNotificationOpts{Repo: "octocat/hello-world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConvertGitHubSubjectType(t *testing.T) {
	tests := []struct {
		input string
		want  forge.NotificationSubjectType
	}{
		{"Issue", forge.NotificationSubjectIssue},
		{"PullRequest", forge.NotificationSubjectPullRequest},
		{"Commit", forge.NotificationSubjectCommit},
		{"Release", forge.NotificationSubjectRelease},
		{"Discussion", forge.NotificationSubjectDiscussion},
		{"RepositoryVulnerabilityAlert", forge.NotificationSubjectRepository},
	}

	for _, tt := range tests {
		got := convertGitHubSubjectType(tt.input)
		if got != tt.want {
			t.Errorf("convertGitHubSubjectType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

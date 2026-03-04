package forges

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v82/github"
)

func newTestGitHubIssueService(srv *httptest.Server) *gitHubIssueService {
	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	return &gitHubIssueService{client: c}
}

func TestGitHubGetIssue(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/issues/42", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(github.Issue{
			Number:  ptrInt(42),
			Title:   ptr("Bug report"),
			Body:    ptr("Something is broken"),
			State:   ptr("open"),
			HTMLURL: ptr("https://github.com/octocat/hello-world/issues/42"),
			Locked:  ptrBool(false),
			User: &github.User{
				Login:     ptr("octocat"),
				AvatarURL: ptr("https://avatars.githubusercontent.com/u/1"),
				HTMLURL:   ptr("https://github.com/octocat"),
			},
			Assignees: []*github.User{
				{Login: ptr("assignee1"), AvatarURL: ptr(""), HTMLURL: ptr("")},
			},
			Labels: []*github.Label{
				{Name: ptr("bug"), Color: ptr("d73a4a"), Description: ptr("Something isn't working")},
			},
			Milestone: &github.Milestone{
				Title:       ptr("v1.0"),
				Number:      ptrInt(1),
				Description: ptr("First release"),
				State:       ptr("open"),
			},
			Comments:  ptrInt(3),
			CreatedAt: &github.Timestamp{Time: parseTime("2024-01-01T00:00:00Z")},
			UpdatedAt: &github.Timestamp{Time: parseTime("2024-01-02T00:00:00Z")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubIssueService(srv)
	issue, err := s.Get(context.Background(), "octocat", "hello-world", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqualInt(t, "Number", 42, issue.Number)
	assertEqual(t, "Title", "Bug report", issue.Title)
	assertEqual(t, "Body", "Something is broken", issue.Body)
	assertEqual(t, "State", "open", issue.State)
	assertEqual(t, "Author.Login", "octocat", issue.Author.Login)
	assertEqualBool(t, "Locked", false, issue.Locked)
	assertEqualInt(t, "Comments", 3, issue.Comments)
	assertEqual(t, "HTMLURL", "https://github.com/octocat/hello-world/issues/42", issue.HTMLURL)

	if len(issue.Assignees) != 1 {
		t.Fatalf("expected 1 assignee, got %d", len(issue.Assignees))
	}
	assertEqual(t, "Assignees[0].Login", "assignee1", issue.Assignees[0].Login)

	if len(issue.Labels) != 1 {
		t.Fatalf("expected 1 label, got %d", len(issue.Labels))
	}
	assertEqual(t, "Labels[0].Name", "bug", issue.Labels[0].Name)
	assertEqual(t, "Labels[0].Color", "d73a4a", issue.Labels[0].Color)

	if issue.Milestone == nil {
		t.Fatal("expected milestone")
	}
	assertEqual(t, "Milestone.Title", "v1.0", issue.Milestone.Title)
	assertEqualInt(t, "Milestone.Number", 1, issue.Milestone.Number)
}

func TestGitHubGetIssueNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/issues/999", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubIssueService(srv)
	_, err := s.Get(context.Background(), "octocat", "hello-world", 999)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGitHubListIssues(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/issues", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.Issue{
			{
				Number: ptrInt(1),
				Title:  ptr("First issue"),
				State:  ptr("open"),
				User:   &github.User{Login: ptr("octocat")},
			},
			{
				Number: ptrInt(2),
				Title:  ptr("Second issue"),
				State:  ptr("open"),
				User:   &github.User{Login: ptr("octocat")},
			},
			{
				// This is a PR, should be filtered out
				Number:           ptrInt(3),
				Title:            ptr("A pull request"),
				State:            ptr("open"),
				User:             &github.User{Login: ptr("octocat")},
				PullRequestLinks: &github.PullRequestLinks{URL: ptr("https://api.github.com/repos/octocat/hello-world/pulls/3")},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubIssueService(srv)
	issues, err := s.List(context.Background(), "octocat", "hello-world", ListIssueOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues (PRs filtered), got %d", len(issues))
	}
	assertEqual(t, "issues[0].Title", "First issue", issues[0].Title)
	assertEqual(t, "issues[1].Title", "Second issue", issues[1].Title)
}

func TestGitHubCreateIssue(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello-world/issues", func(w http.ResponseWriter, r *http.Request) {
		var req github.IssueRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(github.Issue{
			Number:  ptrInt(10),
			Title:   req.Title,
			Body:    req.Body,
			State:   ptr("open"),
			HTMLURL: ptr("https://github.com/octocat/hello-world/issues/10"),
			User:    &github.User{Login: ptr("octocat")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubIssueService(srv)
	issue, err := s.Create(context.Background(), "octocat", "hello-world", CreateIssueOpts{
		Title: "New bug",
		Body:  "Details here",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualInt(t, "Number", 10, issue.Number)
	assertEqual(t, "Title", "New bug", issue.Title)
	assertEqual(t, "State", "open", issue.State)
}

func TestGitHubCloseIssue(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v3/repos/octocat/hello-world/issues/42", func(w http.ResponseWriter, r *http.Request) {
		var req github.IssueRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.GetState() != "closed" {
			t.Errorf("expected state=closed, got %s", req.GetState())
		}
		_ = json.NewEncoder(w).Encode(github.Issue{
			Number: ptrInt(42),
			State:  ptr("closed"),
			User:   &github.User{Login: ptr("octocat")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubIssueService(srv)
	if err := s.Close(context.Background(), "octocat", "hello-world", 42); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubReopenIssue(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v3/repos/octocat/hello-world/issues/42", func(w http.ResponseWriter, r *http.Request) {
		var req github.IssueRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.GetState() != "open" {
			t.Errorf("expected state=open, got %s", req.GetState())
		}
		_ = json.NewEncoder(w).Encode(github.Issue{
			Number: ptrInt(42),
			State:  ptr("open"),
			User:   &github.User{Login: ptr("octocat")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubIssueService(srv)
	if err := s.Reopen(context.Background(), "octocat", "hello-world", 42); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubDeleteIssue(t *testing.T) {
	s := &gitHubIssueService{}
	err := s.Delete(context.Background(), "octocat", "hello-world", 42)
	if err != ErrNotSupported {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}
}

func TestGitHubCreateComment(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello-world/issues/42/comments", func(w http.ResponseWriter, r *http.Request) {
		var req github.IssueComment
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(github.IssueComment{
			ID:      ptrInt64(100),
			Body:    req.Body,
			HTMLURL: ptr("https://github.com/octocat/hello-world/issues/42#issuecomment-100"),
			User:    &github.User{Login: ptr("octocat")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubIssueService(srv)
	comment, err := s.CreateComment(context.Background(), "octocat", "hello-world", 42, "Nice fix!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comment.ID != 100 {
		t.Errorf("expected ID=100, got %d", comment.ID)
	}
	assertEqual(t, "Body", "Nice fix!", comment.Body)
	assertEqual(t, "Author.Login", "octocat", comment.Author.Login)
}

func TestGitHubListComments(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/issues/42/comments", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.IssueComment{
			{
				ID:   ptrInt64(1),
				Body: ptr("First comment"),
				User: &github.User{Login: ptr("alice")},
			},
			{
				ID:   ptrInt64(2),
				Body: ptr("Second comment"),
				User: &github.User{Login: ptr("bob")},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubIssueService(srv)
	comments, err := s.ListComments(context.Background(), "octocat", "hello-world", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	assertEqual(t, "comments[0].Body", "First comment", comments[0].Body)
	assertEqual(t, "comments[0].Author.Login", "alice", comments[0].Author.Login)
	assertEqual(t, "comments[1].Body", "Second comment", comments[1].Body)
}

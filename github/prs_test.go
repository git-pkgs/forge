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

func newTestGitHubPRService(srv *httptest.Server) *gitHubPRService {
	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	return &gitHubPRService{client: c}
}

func TestGitHubGetPR(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(github.PullRequest{
			Number:    ptrInt(1),
			Title:     ptr("Add feature"),
			Body:      ptr("This adds a new feature"),
			State:     ptr("open"),
			Draft:     ptrBool(false),
			Merged:    ptrBool(false),
			Mergeable: ptrBool(true),
			HTMLURL:   ptr("https://github.com/octocat/hello-world/pull/1"),
			DiffURL:   ptr("https://github.com/octocat/hello-world/pull/1.diff"),
			User: &github.User{
				Login:     ptr("octocat"),
				AvatarURL: ptr("https://avatars.githubusercontent.com/u/1"),
				HTMLURL:   ptr("https://github.com/octocat"),
			},
			Head: &github.PullRequestBranch{Ref: ptr("feature-branch")},
			Base: &github.PullRequestBranch{Ref: ptr("main")},
			RequestedReviewers: []*github.User{
				{Login: ptr("reviewer1")},
			},
			Labels: []*github.Label{
				{Name: ptr("enhancement"), Color: ptr("a2eeef")},
			},
			Comments:     ptrInt(2),
			Additions:    ptrInt(10),
			Deletions:    ptrInt(3),
			ChangedFiles: ptrInt(2),
			CreatedAt:    &github.Timestamp{Time: parseTime("2024-01-01T00:00:00Z")},
			UpdatedAt:    &github.Timestamp{Time: parseTime("2024-01-02T00:00:00Z")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubPRService(srv)
	pr, err := s.Get(context.Background(), "octocat", "hello-world", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqualInt(t, "Number", 1, pr.Number)
	assertEqual(t, "Title", "Add feature", pr.Title)
	assertEqual(t, "Body", "This adds a new feature", pr.Body)
	assertEqual(t, "State", "open", pr.State)
	assertEqualBool(t, "Draft", false, pr.Draft)
	assertEqualBool(t, "Merged", false, pr.Merged)
	assertEqualBool(t, "Mergeable", true, pr.Mergeable)
	assertEqual(t, "Head", "feature-branch", pr.Head)
	assertEqual(t, "Base", "main", pr.Base)
	assertEqual(t, "Author.Login", "octocat", pr.Author.Login)
	assertEqualInt(t, "Comments", 2, pr.Comments)
	assertEqualInt(t, "Additions", 10, pr.Additions)
	assertEqualInt(t, "Deletions", 3, pr.Deletions)
	assertEqualInt(t, "ChangedFiles", 2, pr.ChangedFiles)

	if len(pr.Reviewers) != 1 {
		t.Fatalf("expected 1 reviewer, got %d", len(pr.Reviewers))
	}
	assertEqual(t, "Reviewers[0].Login", "reviewer1", pr.Reviewers[0].Login)

	if len(pr.Labels) != 1 {
		t.Fatalf("expected 1 label, got %d", len(pr.Labels))
	}
	assertEqual(t, "Labels[0].Name", "enhancement", pr.Labels[0].Name)
}

func TestGitHubGetPRNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/pulls/999", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubPRService(srv)
	_, err := s.Get(context.Background(), "octocat", "hello-world", 999)
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

func TestGitHubGetPRMerged(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/pulls/5", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(github.PullRequest{
			Number: ptrInt(5),
			Title:  ptr("Merged PR"),
			State:  ptr("closed"),
			Merged: ptrBool(true),
			User:   &github.User{Login: ptr("octocat")},
			Head:   &github.PullRequestBranch{Ref: ptr("feature")},
			Base:   &github.PullRequestBranch{Ref: ptr("main")},
			MergedBy: &github.User{
				Login: ptr("merger"),
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubPRService(srv)
	pr, err := s.Get(context.Background(), "octocat", "hello-world", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "State", "merged", pr.State)
	assertEqualBool(t, "Merged", true, pr.Merged)
	if pr.MergedBy == nil {
		t.Fatal("expected MergedBy")
	}
	assertEqual(t, "MergedBy.Login", "merger", pr.MergedBy.Login)
}

func TestGitHubListPRs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/pulls", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.PullRequest{
			{
				Number: ptrInt(1),
				Title:  ptr("First PR"),
				State:  ptr("open"),
				User:   &github.User{Login: ptr("alice")},
				Head:   &github.PullRequestBranch{Ref: ptr("feature-1")},
				Base:   &github.PullRequestBranch{Ref: ptr("main")},
			},
			{
				Number: ptrInt(2),
				Title:  ptr("Second PR"),
				State:  ptr("open"),
				User:   &github.User{Login: ptr("bob")},
				Head:   &github.PullRequestBranch{Ref: ptr("feature-2")},
				Base:   &github.PullRequestBranch{Ref: ptr("main")},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubPRService(srv)
	prs, err := s.List(context.Background(), "octocat", "hello-world", forge.ListPROpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}
	assertEqual(t, "prs[0].Title", "First PR", prs[0].Title)
	assertEqual(t, "prs[0].Head", "feature-1", prs[0].Head)
	assertEqual(t, "prs[1].Title", "Second PR", prs[1].Title)
}

func TestGitHubCreatePR(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello-world/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(github.PullRequest{
			Number:  ptrInt(10),
			Title:   ptr("New feature"),
			State:   ptr("open"),
			HTMLURL: ptr("https://github.com/octocat/hello-world/pull/10"),
			User:    &github.User{Login: ptr("octocat")},
			Head:    &github.PullRequestBranch{Ref: ptr("my-branch")},
			Base:    &github.PullRequestBranch{Ref: ptr("main")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubPRService(srv)
	pr, err := s.Create(context.Background(), "octocat", "hello-world", forge.CreatePROpts{
		Title: "New feature",
		Body:  "Description",
		Head:  "my-branch",
		Base:  "main",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualInt(t, "Number", 10, pr.Number)
	assertEqual(t, "Title", "New feature", pr.Title)
	assertEqual(t, "State", "open", pr.State)
}

func TestGitHubClosePR(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v3/repos/octocat/hello-world/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(github.PullRequest{
			Number: ptrInt(1),
			State:  ptr("closed"),
			User:   &github.User{Login: ptr("octocat")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubPRService(srv)
	if err := s.Close(context.Background(), "octocat", "hello-world", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubReopenPR(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/v3/repos/octocat/hello-world/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(github.PullRequest{
			Number: ptrInt(1),
			State:  ptr("open"),
			User:   &github.User{Login: ptr("octocat")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubPRService(srv)
	if err := s.Reopen(context.Background(), "octocat", "hello-world", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubMergePR(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v3/repos/octocat/hello-world/pulls/1/merge", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(github.PullRequestMergeResult{
			Merged: ptrBool(true),
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubPRService(srv)
	if err := s.Merge(context.Background(), "octocat", "hello-world", 1, forge.MergePROpts{Method: "squash"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubDiffPR(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") == "application/vnd.github.v3.diff" {
			_, _ = w.Write([]byte("diff --git a/file.txt b/file.txt\n"))
			return
		}
		_ = json.NewEncoder(w).Encode(github.PullRequest{Number: ptrInt(1)})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubPRService(srv)
	diff, err := s.Diff(context.Background(), "octocat", "hello-world", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff == "" {
		t.Error("expected non-empty diff")
	}
}

func TestGitHubPRCreateComment(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello-world/issues/1/comments", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(github.IssueComment{
			ID:   ptrInt64(200),
			Body: ptr("LGTM"),
			User: &github.User{Login: ptr("reviewer")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubPRService(srv)
	comment, err := s.CreateComment(context.Background(), "octocat", "hello-world", 1, "LGTM")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comment.ID != 200 {
		t.Errorf("expected ID=200, got %d", comment.ID)
	}
	assertEqual(t, "Body", "LGTM", comment.Body)
	assertEqual(t, "Author.Login", "reviewer", comment.Author.Login)
}

func TestGitHubPRListComments(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/issues/1/comments", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.IssueComment{
			{
				ID:   ptrInt64(1),
				Body: ptr("First"),
				User: &github.User{Login: ptr("alice")},
			},
			{
				ID:   ptrInt64(2),
				Body: ptr("Second"),
				User: &github.User{Login: ptr("bob")},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubPRService(srv)
	comments, err := s.ListComments(context.Background(), "octocat", "hello-world", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	assertEqual(t, "comments[0].Body", "First", comments[0].Body)
	assertEqual(t, "comments[1].Body", "Second", comments[1].Body)
}

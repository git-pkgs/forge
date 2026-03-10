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

func newTestGitHubReviewService(srv *httptest.Server) *gitHubReviewService {
	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	return &gitHubReviewService{client: c}
}

func TestGitHubListReviews(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/pulls/1/reviews", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.PullRequestReview{
			{
				ID:          ptrInt64(100),
				State:       ptr("APPROVED"),
				Body:        ptr("Looks good!"),
				User:        &github.User{Login: ptr("alice")},
				SubmittedAt: &github.Timestamp{Time: parseTime("2024-01-15T10:00:00Z")},
			},
			{
				ID:    ptrInt64(101),
				State: ptr("CHANGES_REQUESTED"),
				Body:  ptr("Please fix the tests"),
				User:  &github.User{Login: ptr("bob")},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubReviewService(srv)
	reviews, err := s.List(context.Background(), "octocat", "hello-world", 1, forge.ListReviewOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reviews) != 2 {
		t.Fatalf("expected 2 reviews, got %d", len(reviews))
	}

	assertEqual(t, "reviews[0].State", "approved", string(reviews[0].State))
	assertEqual(t, "reviews[0].Body", "Looks good!", reviews[0].Body)
	assertEqual(t, "reviews[0].Author.Login", "alice", reviews[0].Author.Login)

	assertEqual(t, "reviews[1].State", "changes_requested", string(reviews[1].State))
	assertEqual(t, "reviews[1].Author.Login", "bob", reviews[1].Author.Login)
}

func TestGitHubListReviewsNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/pulls/999/reviews", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubReviewService(srv)
	_, err := s.List(context.Background(), "octocat", "hello-world", 999, forge.ListReviewOpts{})
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

func TestGitHubSubmitReviewApprove(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello-world/pulls/1/reviews", func(w http.ResponseWriter, r *http.Request) {
		var req github.PullRequestReviewRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.GetEvent() != "APPROVE" {
			t.Errorf("expected event APPROVE, got %s", req.GetEvent())
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(github.PullRequestReview{
			ID:    ptrInt64(200),
			State: ptr("APPROVED"),
			Body:  ptr("LGTM"),
			User:  &github.User{Login: ptr("reviewer")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubReviewService(srv)
	review, err := s.Submit(context.Background(), "octocat", "hello-world", 1, forge.SubmitReviewOpts{
		State: forge.ReviewApproved,
		Body:  "LGTM",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "State", "approved", string(review.State))
	assertEqual(t, "Author.Login", "reviewer", review.Author.Login)
}

func TestGitHubSubmitReviewRequestChanges(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello-world/pulls/1/reviews", func(w http.ResponseWriter, r *http.Request) {
		var req github.PullRequestReviewRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.GetEvent() != "REQUEST_CHANGES" {
			t.Errorf("expected event REQUEST_CHANGES, got %s", req.GetEvent())
		}
		_ = json.NewEncoder(w).Encode(github.PullRequestReview{
			ID:    ptrInt64(201),
			State: ptr("CHANGES_REQUESTED"),
			Body:  ptr("Fix the tests"),
			User:  &github.User{Login: ptr("reviewer")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubReviewService(srv)
	review, err := s.Submit(context.Background(), "octocat", "hello-world", 1, forge.SubmitReviewOpts{
		State: forge.ReviewChangesRequested,
		Body:  "Fix the tests",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "State", "changes_requested", string(review.State))
}

func TestGitHubRequestReviewers(t *testing.T) {
	var requested github.ReviewersRequest
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello-world/pulls/1/requested_reviewers", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&requested)
		_ = json.NewEncoder(w).Encode(github.PullRequest{
			Number: ptrInt(1),
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubReviewService(srv)
	err := s.RequestReviewers(context.Background(), "octocat", "hello-world", 1, []string{"alice", "bob"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertSliceEqual(t, "Reviewers", []string{"alice", "bob"}, requested.Reviewers)
}

func TestGitHubRemoveReviewers(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v3/repos/octocat/hello-world/pulls/1/requested_reviewers", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubReviewService(srv)
	err := s.RemoveReviewers(context.Background(), "octocat", "hello-world", 1, []string{"alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

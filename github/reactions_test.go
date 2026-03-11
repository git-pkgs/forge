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

func TestGitHubListIssueCommentReactions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/issues/comments/42/reactions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.Reaction{
			{
				ID:      ptrInt64(1),
				Content: ptr("+1"),
				User:    &github.User{Login: ptr("alice")},
			},
			{
				ID:      ptrInt64(2),
				Content: ptr("heart"),
				User:    &github.User{Login: ptr("bob")},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubIssueService(srv)
	reactions, err := s.ListReactions(context.Background(), "octocat", "hello-world", 1, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reactions) != 2 {
		t.Fatalf("expected 2 reactions, got %d", len(reactions))
	}

	assertEqual(t, "reactions[0].Content", "+1", reactions[0].Content)
	assertEqual(t, "reactions[0].User", "alice", reactions[0].User)
	assertEqual(t, "reactions[1].Content", "heart", reactions[1].Content)
	assertEqual(t, "reactions[1].User", "bob", reactions[1].User)
}

func TestGitHubAddIssueCommentReaction(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello-world/issues/comments/42/reactions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(&github.Reaction{
			ID:      ptrInt64(10),
			Content: ptr("rocket"),
			User:    &github.User{Login: ptr("alice")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubIssueService(srv)
	reaction, err := s.AddReaction(context.Background(), "octocat", "hello-world", 1, 42, "rocket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Content", "rocket", reaction.Content)
	assertEqual(t, "User", "alice", reaction.User)
}

func TestGitHubListIssueCommentReactionsNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/issues/comments/999/reactions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubIssueService(srv)
	_, err := s.ListReactions(context.Background(), "octocat", "hello-world", 1, 999)
	if err != forge.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGitHubPRListCommentReactions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/issues/comments/50/reactions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*github.Reaction{
			{
				ID:      ptrInt64(5),
				Content: ptr("eyes"),
				User:    &github.User{Login: ptr("carol")},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubPRService(srv)
	reactions, err := s.ListReactions(context.Background(), "octocat", "hello-world", 10, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reactions) != 1 {
		t.Fatalf("expected 1 reaction, got %d", len(reactions))
	}
	assertEqual(t, "Content", "eyes", reactions[0].Content)
	assertEqual(t, "User", "carol", reactions[0].User)
}

func TestGitHubPRAddCommentReaction(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello-world/issues/comments/50/reactions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(&github.Reaction{
			ID:      ptrInt64(11),
			Content: ptr("laugh"),
			User:    &github.User{Login: ptr("carol")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	s := newTestGitHubPRService(srv)
	reaction, err := s.AddReaction(context.Background(), "octocat", "hello-world", 10, 50, "laugh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "Content", "laugh", reaction.Content)
}

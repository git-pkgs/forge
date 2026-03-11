package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGiteaListIssueCommentReactions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/issues/comments/42/reactions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"user":    map[string]any{"login": "alice"},
				"content": "+1",
			},
			{
				"user":    map[string]any{"login": "bob"},
				"content": "heart",
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	reactions, err := f.Issues().ListReactions(context.Background(), "testorg", "testrepo", 1, 42)
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

func TestGiteaAddIssueCommentReaction(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("POST /api/v1/repos/testorg/testrepo/issues/comments/42/reactions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user":    map[string]any{"login": "alice"},
			"content": "rocket",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	reaction, err := f.Issues().AddReaction(context.Background(), "testorg", "testrepo", 1, 42, "rocket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Content", "rocket", reaction.Content)
	assertEqual(t, "User", "alice", reaction.User)
}

func TestGiteaListIssueCommentReactionsNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/issues/comments/999/reactions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Issues().ListReactions(context.Background(), "testorg", "testrepo", 1, 999)
	if err != forge.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGiteaPRListCommentReactions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/issues/comments/50/reactions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"user":    map[string]any{"login": "carol"},
				"content": "eyes",
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	reactions, err := f.PullRequests().ListReactions(context.Background(), "testorg", "testrepo", 10, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reactions) != 1 {
		t.Fatalf("expected 1 reaction, got %d", len(reactions))
	}
	assertEqual(t, "Content", "eyes", reactions[0].Content)
	assertEqual(t, "User", "carol", reactions[0].User)
}

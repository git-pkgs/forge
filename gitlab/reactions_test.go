package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGitLabListIssueCommentReactions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo/issues/1/notes/42/award_emoji", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":   1,
				"name": "thumbsup",
				"user": map[string]any{"username": "alice", "id": 10},
			},
			{
				"id":   2,
				"name": "heart",
				"user": map[string]any{"username": "bob", "id": 11},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	reactions, err := f.Issues().ListReactions(context.Background(), "mygroup", "myrepo", 1, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reactions) != 2 {
		t.Fatalf("expected 2 reactions, got %d", len(reactions))
	}

	assertEqual(t, "reactions[0].Content", "thumbsup", reactions[0].Content)
	assertEqual(t, "reactions[0].User", "alice", reactions[0].User)
	assertEqual(t, "reactions[1].Content", "heart", reactions[1].Content)
	assertEqual(t, "reactions[1].User", "bob", reactions[1].User)
}

func TestGitLabAddIssueCommentReaction(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v4/projects/mygroup%2Fmyrepo/issues/1/notes/42/award_emoji", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   10,
			"name": "rocket",
			"user": map[string]any{"username": "alice", "id": 10},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	reaction, err := f.Issues().AddReaction(context.Background(), "mygroup", "myrepo", 1, 42, "rocket")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Content", "rocket", reaction.Content)
	assertEqual(t, "User", "alice", reaction.User)
}

func TestGitLabListMRCommentReactions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo/merge_requests/5/notes/50/award_emoji", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":   3,
				"name": "eyes",
				"user": map[string]any{"username": "carol", "id": 12},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	reactions, err := f.PullRequests().ListReactions(context.Background(), "mygroup", "myrepo", 5, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reactions) != 1 {
		t.Fatalf("expected 1 reaction, got %d", len(reactions))
	}
	assertEqual(t, "Content", "eyes", reactions[0].Content)
	assertEqual(t, "User", "carol", reactions[0].User)
}

func TestGitLabAddMRCommentReaction(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v4/projects/mygroup%2Fmyrepo/merge_requests/5/notes/50/award_emoji", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   11,
			"name": "laugh",
			"user": map[string]any{"username": "carol", "id": 12},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	reaction, err := f.PullRequests().AddReaction(context.Background(), "mygroup", "myrepo", 5, 50, "laugh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "Content", "laugh", reaction.Content)
}

func TestGitLabListReactionsNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo/issues/1/notes/999/award_emoji", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Issues().ListReactions(context.Background(), "mygroup", "myrepo", 1, 999)
	if err != forge.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

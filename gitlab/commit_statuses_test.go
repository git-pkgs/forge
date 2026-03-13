package gitlab

import (
	"context"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitLabListCommitStatuses(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo/repository/commits/abc123/statuses", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":          1,
				"status":      "success",
				"name":        "ci/tests",
				"description": "All tests passed",
				"target_url":  "https://ci.example.com/1",
				"author":      map[string]any{"username": "bot"},
			},
			{
				"id":          2,
				"status":      "pending",
				"name":        "ci/lint",
				"description": "Running",
				"author":      map[string]any{"username": "bot"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	statuses, err := f.CommitStatuses().List(context.Background(), "mygroup", "myrepo", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}
	assertEqual(t, "statuses[0].State", "success", statuses[0].State)
	assertEqual(t, "statuses[0].Context", "ci/tests", statuses[0].Context)
	assertEqual(t, "statuses[0].Description", "All tests passed", statuses[0].Description)
	assertEqual(t, "statuses[0].Creator", "bot", statuses[0].Creator)
	assertEqual(t, "statuses[1].State", "pending", statuses[1].State)
}

func TestGitLabSetCommitStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v4/projects/mygroup%2Fmyrepo/statuses/abc123", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          1,
			"status":      "success",
			"name":        "my-check",
			"description": "Everything passed",
			"target_url":  "https://example.com",
			"author":      map[string]any{"username": "bot"},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	status, err := f.CommitStatuses().Set(context.Background(), "mygroup", "myrepo", "abc123", forge.SetCommitStatusOpts{
		State:       "success",
		Context:     "my-check",
		Description: "Everything passed",
		TargetURL:   "https://example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "State", "success", status.State)
	assertEqual(t, "Context", "my-check", status.Context)
	assertEqual(t, "Description", "Everything passed", status.Description)
}

func TestGitLabListCommitStatusesNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fnope/repository/commits/abc123/statuses", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "404 Project Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.CommitStatuses().List(context.Background(), "mygroup", "nope", "abc123")
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

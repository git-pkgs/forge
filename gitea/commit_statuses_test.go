package gitea

import (
	"context"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGiteaListCommitStatuses(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/commits/abc123/statuses", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":          1,
				"status":      "success",
				"context":     "ci/tests",
				"description": "All tests passed",
				"target_url":  "https://ci.example.com/1",
				"creator":     map[string]any{"login": "bot"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	statuses, err := f.CommitStatuses().List(context.Background(), "testorg", "testrepo", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	assertEqual(t, "statuses[0].State", "success", statuses[0].State)
	assertEqual(t, "statuses[0].Context", "ci/tests", statuses[0].Context)
	assertEqual(t, "statuses[0].Creator", "bot", statuses[0].Creator)
}

func TestGiteaSetCommitStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("POST /api/v1/repos/testorg/testrepo/statuses/abc123", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          1,
			"status":      "success",
			"context":     "my-check",
			"description": "Everything passed",
			"target_url":  "https://example.com",
			"creator":     map[string]any{"login": "bot"},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	status, err := f.CommitStatuses().Set(context.Background(), "testorg", "testrepo", "abc123", forge.SetCommitStatusOpts{
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
}

func TestGiteaListCommitStatusesNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/nope/commits/abc123/statuses", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.CommitStatuses().List(context.Background(), "testorg", "nope", "abc123")
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	forge "github.com/git-pkgs/forge"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func giteaVersionHandler125(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, `{"version":"1.25.0"}`)
}

func TestGiteaCIListRuns(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler125)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/actions/runs", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_count": 1,
			"workflow_runs": []map[string]any{
				{
					"id":            42,
					"display_title": "CI Pipeline",
					"status":        "completed",
					"conclusion":    "success",
					"head_branch":   "main",
					"head_sha":      "abc123",
					"event":         "push",
					"html_url":      "https://codeberg.org/testorg/testrepo/actions/runs/42",
					"actor": map[string]any{
						"login":      "testuser",
						"avatar_url": "https://codeberg.org/avatars/1",
					},
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	runs, err := f.CI().ListRuns(context.Background(), "testorg", "testrepo", forge.ListCIRunOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	r := runs[0]
	if r.ID != 42 {
		t.Errorf("ID: want 42, got %d", r.ID)
	}
	assertEqual(t, "Title", "CI Pipeline", r.Title)
	assertEqual(t, "Status", "completed", r.Status)
	assertEqual(t, "Conclusion", "success", r.Conclusion)
	assertEqual(t, "Branch", "main", r.Branch)
	assertEqual(t, "SHA", "abc123", r.SHA)
	assertEqual(t, "Event", "push", r.Event)
	assertEqual(t, "Author.Login", "testuser", r.Author.Login)
}

func TestGiteaCIListRunsWithFilters(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler125)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/actions/runs", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("branch") != "develop" {
			t.Errorf("expected branch=develop, got %q", q.Get("branch"))
		}
		if q.Get("status") != "running" {
			t.Errorf("expected status=running, got %q", q.Get("status"))
		}
		if q.Get("actor") != "testuser" {
			t.Errorf("expected actor=testuser, got %q", q.Get("actor"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_count":   0,
			"workflow_runs": []map[string]any{},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	runs, err := f.CI().ListRuns(context.Background(), "testorg", "testrepo", forge.ListCIRunOpts{
		Branch: "develop",
		Status: "running",
		User:   "testuser",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("expected 0 runs, got %d", len(runs))
	}
}

func TestGiteaCIGetRun(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler125)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/actions/runs/42", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":            42,
			"display_title": "CI Pipeline",
			"status":        "completed",
			"conclusion":    "success",
			"head_branch":   "main",
			"head_sha":      "abc123",
		})
	})
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/actions/runs/42/jobs", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_count": 1,
			"jobs": []map[string]any{
				{
					"id":         100,
					"name":       "build",
					"status":     "completed",
					"conclusion": "success",
					"html_url":   "https://codeberg.org/testorg/testrepo/actions/runs/42/jobs/100",
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	run, err := f.CI().GetRun(context.Background(), "testorg", "testrepo", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.ID != 42 {
		t.Errorf("ID: want 42, got %d", run.ID)
	}
	assertEqual(t, "Title", "CI Pipeline", run.Title)
	assertEqual(t, "Conclusion", "success", run.Conclusion)
	if len(run.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(run.Jobs))
	}
	assertEqual(t, "Jobs[0].Name", "build", run.Jobs[0].Name)
	assertEqual(t, "Jobs[0].Status", "completed", run.Jobs[0].Status)
}

func TestGiteaCIGetRunNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler125)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/actions/runs/999", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.CI().GetRun(context.Background(), "testorg", "testrepo", 999)
	if err != forge.ErrNotFound {
		t.Fatalf("expected forge.ErrNotFound, got %v", err)
	}
}

func TestGiteaCIGetJobLog(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler125)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/actions/jobs/100/logs", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "Build started\nStep 1: compile\nBuild finished")
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	rc, err := f.CI().GetJobLog(context.Background(), "testorg", "testrepo", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	assertEqual(t, "log content", "Build started\nStep 1: compile\nBuild finished", string(data))
}

func TestGiteaCITriggerRunNotSupported(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler125)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	err := f.CI().TriggerRun(context.Background(), "testorg", "testrepo", forge.TriggerCIRunOpts{})
	if err != forge.ErrNotSupported {
		t.Fatalf("expected forge.ErrNotSupported, got %v", err)
	}
}

func TestGiteaCICancelRunNotSupported(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler125)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	err := f.CI().CancelRun(context.Background(), "testorg", "testrepo", 42)
	if err != forge.ErrNotSupported {
		t.Fatalf("expected forge.ErrNotSupported, got %v", err)
	}
}

func TestGiteaCIRetryRunNotSupported(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler125)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	err := f.CI().RetryRun(context.Background(), "testorg", "testrepo", 42)
	if err != forge.ErrNotSupported {
		t.Fatalf("expected forge.ErrNotSupported, got %v", err)
	}
}

package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	forge "github.com/git-pkgs/forge"
)

const statusTestSHA = "fd61ca641c0db3ab42ad10bc1b2b7190033aee77"

// newStatusTestForge points the Bitbucket API at a test server running h.
func newStatusTestForge(t *testing.T, h http.Handler) (forge.Forge, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	t.Cleanup(func() { setBitbucketAPI(origAPI) })
	return New("test-token", srv.Client()), srv
}

func TestBitbucketListCommitStatuses(t *testing.T) {
	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/commit/"+statusTestSHA+"/statuses", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want bearer token", got)
		}
		if r.URL.Query().Get("page") == "" {
			if got := r.URL.Query().Get("pagelen"); got != "100" {
				t.Errorf("pagelen = %q, want 100", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{{
					"key":         "default",
					"name":        "Pipeline - default",
					"state":       "FAILED",
					"description": "2 tests failed",
					"url":         "https://bitbucket.org/atlassian/stash/pipelines/results/2865",
					"created_on":  "2026-04-30T04:38:20.636159+00:00",
				}},
				"next": srvURL + "/2.0/repositories/atlassian/stash/commit/" + statusTestSHA + "/statuses?pagelen=100&page=2",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{"key": "lint", "state": "SUCCESSFUL"},
				{"key": "deploy", "state": "INPROGRESS"},
				{"key": "e2e", "state": "STOPPED"},
			},
		})
	})
	f, srv := newStatusTestForge(t, mux)
	srvURL = srv.URL

	statuses, err := f.CommitStatuses().List(context.Background(), "atlassian", "stash", statusTestSHA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []forge.CommitStatus{
		{
			State:       forge.CommitStatusFailure,
			Context:     "default",
			Description: "2 tests failed",
			TargetURL:   "https://bitbucket.org/atlassian/stash/pipelines/results/2865",
			CreatedAt:   time.Date(2026, 4, 30, 4, 38, 20, 636159000, time.UTC),
		},
		{State: forge.CommitStatusSuccess, Context: "lint"},
		{State: forge.CommitStatusPending, Context: "deploy"},
		{State: forge.CommitStatusCancelled, Context: "e2e"},
	}
	if len(statuses) != len(want) {
		t.Fatalf("got %d statuses, want %d: %+v", len(statuses), len(want), statuses)
	}
	for i := range want {
		got := statuses[i]
		if got.State != want[i].State || got.Context != want[i].Context ||
			got.Description != want[i].Description || got.TargetURL != want[i].TargetURL ||
			!got.CreatedAt.Equal(want[i].CreatedAt) {
			t.Errorf("status %d = %+v, want %+v", i, got, want[i])
		}
	}
}

func TestBitbucketListCommitStatusesNotFound(t *testing.T) {
	f, _ := newStatusTestForge(t, http.NotFoundHandler())
	_, err := f.CommitStatuses().List(context.Background(), "atlassian", "stash", "missing")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestBitbucketSetCommitStatus(t *testing.T) {
	var body map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("POST /2.0/repositories/atlassian/stash/commit/"+statusTestSHA+"/statuses/build", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key":         "ci/test",
			"name":        "ci/test",
			"state":       "SUCCESSFUL",
			"description": "42 tests passed",
			"url":         "https://ci.example.com/build/1",
			"created_on":  "2026-11-19T10:00:00+00:00",
		})
	})
	f, _ := newStatusTestForge(t, mux)

	status, err := f.CommitStatuses().Set(context.Background(), "atlassian", "stash", statusTestSHA, forge.SetCommitStatusOpts{
		State:       "success",
		Context:     "ci/test",
		Description: "42 tests passed",
		TargetURL:   "https://ci.example.com/build/1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantBody := map[string]any{
		"key":         "ci/test",
		"name":        "ci/test",
		"state":       "SUCCESSFUL",
		"description": "42 tests passed",
		"url":         "https://ci.example.com/build/1",
	}
	if len(body) != len(wantBody) {
		t.Errorf("body = %v, want %v", body, wantBody)
	}
	for k, v := range wantBody {
		if body[k] != v {
			t.Errorf("body[%q] = %v, want %v", k, body[k], v)
		}
	}

	assertEqual(t, "State", "success", status.State)
	assertEqual(t, "Context", "ci/test", status.Context)
	assertEqual(t, "Description", "42 tests passed", status.Description)
	assertEqual(t, "TargetURL", "https://ci.example.com/build/1", status.TargetURL)
	if !status.CreatedAt.Equal(time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("CreatedAt = %v", status.CreatedAt)
	}
}

func TestBitbucketSetCommitStatusOmitsEmptyFields(t *testing.T) {
	var keys []string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /2.0/repositories/atlassian/stash/commit/"+statusTestSHA+"/statuses/build", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		for k := range body {
			keys = append(keys, k)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"key": "lint", "state": "INPROGRESS"})
	})
	f, _ := newStatusTestForge(t, mux)

	if _, err := f.CommitStatuses().Set(context.Background(), "atlassian", "stash", statusTestSHA, forge.SetCommitStatusOpts{
		State:   "pending",
		Context: "lint",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	slices.Sort(keys)
	if want := []string{"key", "name", "state"}; !slices.Equal(keys, want) {
		t.Errorf("body keys = %v, want %v", keys, want)
	}
}

func TestBitbucketCommitStatusState(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"success", "SUCCESSFUL"},
		{"SUCCESSFUL", "SUCCESSFUL"},
		{"failure", "FAILED"},
		{"error", "FAILED"},
		{"FAILED", "FAILED"},
		{"pending", "INPROGRESS"},
		{"INPROGRESS", "INPROGRESS"},
		{"cancelled", "STOPPED"},
		{"stopped", "STOPPED"},
		{"skipped", "SKIPPED"},
		{"bogus", "BOGUS"},
	}
	for _, tt := range tests {
		if got := bitbucketCommitStatusState(tt.in); got != tt.want {
			t.Errorf("bitbucketCommitStatusState(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBitbucketSetCommitStatusRejected(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /2.0/repositories/atlassian/stash/commit/"+statusTestSHA+"/statuses/build", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type": "error", "error": {"message": "state: Value must be one of FAILED, INPROGRESS, STOPPED, SUCCESSFUL"}}`))
	})
	f, _ := newStatusTestForge(t, mux)

	_, err := f.CommitStatuses().Set(context.Background(), "atlassian", "stash", statusTestSHA, forge.SetCommitStatusOpts{
		State:   "skipped",
		Context: "lint",
	})
	var httpErr *forge.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusBadRequest {
		t.Errorf("error = %v, want HTTP 400", err)
	}
}

func TestBitbucketSetCommitStatusReadOnly(t *testing.T) {
	f := New("test-token", forge.ReadOnlyHTTPClient(nil))
	_, err := f.CommitStatuses().Set(context.Background(), "atlassian", "stash", statusTestSHA, forge.SetCommitStatusOpts{
		State:   "success",
		Context: "lint",
	})
	if !errors.Is(err, forge.ErrReadOnly) {
		t.Errorf("error = %v, want ErrReadOnly", err)
	}
}

package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGiteaUpdateIssueReplacesLabels(t *testing.T) {
	var replaceCalled bool
	var replaceBody struct {
		Labels []int64 `json:"labels"`
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/octocat/hello-world/labels", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": 7, "name": "ready-for-agent", "color": "#0e8a16"},
			{"id": 8, "name": "needs-triage", "color": "#ededed"},
		})
	})
	mux.HandleFunc("PUT /api/v1/repos/octocat/hello-world/issues/42/labels", func(w http.ResponseWriter, r *http.Request) {
		replaceCalled = true
		_ = json.NewDecoder(r.Body).Decode(&replaceBody)
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": 7, "name": "ready-for-agent", "color": "#0e8a16"},
		})
	})
	mux.HandleFunc("GET /api/v1/repos/octocat/hello-world/issues/42", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"number": 42,
			"title":  "the issue",
			"state":  "open",
			"labels": []map[string]any{
				{"id": 7, "name": "ready-for-agent", "color": "#0e8a16"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	issue, err := f.Issues().Update(context.Background(), "octocat", "hello-world", 42, forge.UpdateIssueOpts{
		Labels: []string{"ready-for-agent"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !replaceCalled {
		t.Fatal("expected ReplaceIssueLabels (PUT /issues/42/labels) to be called")
	}
	if len(replaceBody.Labels) != 1 || replaceBody.Labels[0] != 7 {
		t.Errorf("expected PUT body labels=[7], got %v", replaceBody.Labels)
	}
	if len(issue.Labels) != 1 || issue.Labels[0].Name != "ready-for-agent" {
		t.Errorf("expected returned issue to have label ready-for-agent, got %+v", issue.Labels)
	}
}

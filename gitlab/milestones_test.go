package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGitLabListMilestonesWithOpenState(t *testing.T) {
	var gotState string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo/milestones", func(w http.ResponseWriter, r *http.Request) {
		gotState = r.URL.Query().Get("state")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":    1,
				"title": "v1.0",
				"state": "active",
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)

	milestones, err := f.Milestones().List(context.Background(), "mygroup", "myrepo", forge.ListMilestoneOpts{
		State: "open",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(milestones) != 1 {
		t.Fatalf("expected 1 milestone, got %d", len(milestones))
	}
	assertEqual(t, "state query param", "active", gotState)
	assertEqual(t, "milestone state", "open", milestones[0].State)
}

package bitbucket

import (
	"context"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBitbucketListForks(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/forks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{
					"full_name":  "alice/stash",
					"name":       "stash",
					"language":   "Java",
					"created_on": "2024-01-01T00:00:00Z",
					"updated_on": "2024-06-01T00:00:00Z",
					"links": map[string]any{
						"html":  map[string]string{"href": "https://bitbucket.org/alice/stash"},
						"clone": []map[string]string{{"href": "https://bitbucket.org/alice/stash.git", "name": "https"}},
					},
					"owner": map[string]any{"nickname": "alice"},
				},
			},
			"next": "",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	oldAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer func() { bitbucketAPI = oldAPI }()

	f := New("token", nil)
	forks, err := f.Repos().ListForks(context.Background(), "atlassian", "stash", forge.ListForksOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(forks) != 1 {
		t.Fatalf("expected 1 fork, got %d", len(forks))
	}
	if forks[0].FullName != "alice/stash" {
		t.Errorf("expected full_name alice/stash, got %s", forks[0].FullName)
	}
}

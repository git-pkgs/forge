package gitea

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"code.gitea.io/sdk/gitea"
)

func TestResolveLabelIDs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/myorg/myrepo/labels", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "name": "bug", "color": "#d73a4a"},
			{"id": 2, "name": "enhancement", "color": "#a2eeef"},
			{"id": 3, "name": "docs", "color": "#0075ca"},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := gitea.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("failed to create gitea client: %v", err)
	}

	ids, err := resolveLabelIDs(client, "myorg", "myrepo", []string{"bug", "docs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d", len(ids))
	}

	// The order depends on iteration through the labels list, not the input order.
	// Both 1 (bug) and 3 (docs) should be present.
	idSet := map[int64]bool{}
	for _, id := range ids {
		idSet[id] = true
	}
	if !idSet[1] {
		t.Errorf("expected ID 1 (bug) in results, got %v", ids)
	}
	if !idSet[3] {
		t.Errorf("expected ID 3 (docs) in results, got %v", ids)
	}
}

func TestResolveLabelIDsMissing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/myorg/myrepo/labels", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "name": "bug", "color": "#d73a4a"},
			{"id": 2, "name": "enhancement", "color": "#a2eeef"},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := gitea.NewClient(srv.URL)
	if err != nil {
		t.Fatalf("failed to create gitea client: %v", err)
	}

	_, err = resolveLabelIDs(client, "myorg", "myrepo", []string{"bug", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for missing label, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to mention 'nonexistent', got: %v", err)
	}
	if !strings.Contains(err.Error(), "labels not found") {
		t.Errorf("expected error to contain 'labels not found', got: %v", err)
	}
}

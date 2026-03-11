package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-github/v82/github"
)

func TestGitHubGetRateLimit(t *testing.T) {
	resetTime := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/rate_limit", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"resources": map[string]any{
				"core": map[string]any{
					"limit":     5000,
					"remaining": 4999,
					"reset":     resetTime.Unix(),
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	f := &gitHubForge{client: c}

	rl, err := f.GetRateLimit(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqualInt(t, "Limit", 5000, rl.Limit)
	assertEqualInt(t, "Remaining", 4999, rl.Remaining)
	if !rl.Reset.Equal(resetTime) {
		t.Errorf("Reset: want %v, got %v", resetTime, rl.Reset)
	}
}

package forges

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// DetectForgeType probes a domain to identify which forge software it runs.
// It checks HTTP response headers first, then falls back to API endpoints.
// If hc is nil, http.DefaultClient is used.
func DetectForgeType(ctx context.Context, domain string, hc ...*http.Client) (ForgeType, error) {
	client := http.DefaultClient
	if len(hc) > 0 && hc[0] != nil {
		client = hc[0]
	}
	baseURL := "https://" + domain

	ft, err := detectFromHeaders(ctx, client, baseURL)
	if err == nil && ft != Unknown {
		return ft, nil
	}

	return detectFromAPI(ctx, client, baseURL)
}

func detectFromHeaders(ctx context.Context, client *http.Client, baseURL string) (ForgeType, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		return Unknown, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return Unknown, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.Header.Get("X-Forgejo-Version") != "" {
		return Forgejo, nil
	}
	if resp.Header.Get("X-Gitea-Version") != "" {
		return Gitea, nil
	}
	if resp.Header.Get("X-Gitlab-Meta") != "" {
		return GitLab, nil
	}
	if resp.Header.Get("X-GitHub-Request-Id") != "" {
		return GitHub, nil
	}

	return Unknown, nil
}

func detectFromAPI(ctx context.Context, client *http.Client, baseURL string) (ForgeType, error) {
	// Try Gitea/Forgejo /api/v1/version
	if ft, err := probeGiteaAPI(ctx, client, baseURL); err == nil {
		return ft, nil
	}

	// Try GitLab /api/v4/version
	if ok, err := probeURL(ctx, client, baseURL+"/api/v4/version"); err == nil && ok {
		return GitLab, nil
	}

	// Try GitHub Enterprise /api/v3/meta
	if ok, err := probeURL(ctx, client, baseURL+"/api/v3/meta"); err == nil && ok {
		return GitHub, nil
	}

	return Unknown, fmt.Errorf("could not detect forge type for %s", baseURL)
}

func probeGiteaAPI(ctx context.Context, client *http.Client, baseURL string) (ForgeType, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/version", nil)
	if err != nil {
		return Unknown, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return Unknown, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return Unknown, fmt.Errorf("status %d", resp.StatusCode)
	}

	var v struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return Unknown, err
	}

	if strings.Contains(strings.ToLower(v.Version), "forgejo") {
		return Forgejo, nil
	}
	return Gitea, nil
}

func probeURL(ctx context.Context, client *http.Client, url string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK, nil
}

package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	forge "github.com/git-pkgs/forge"
)

type giteaRateLimitResponse struct {
	Resources struct {
		Core struct {
			Limit     int   `json:"limit"`
			Remaining int   `json:"remaining"`
			Reset     int64 `json:"reset"`
		} `json:"core"`
	} `json:"resources"`
}

func (f *giteaForge) GetRateLimit(ctx context.Context) (*forge.RateLimit, error) {
	url := f.baseURL + "/api/v1/rate_limit"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if f.token != "" {
		req.Header.Set("Authorization", "token "+f.token)
	}

	hc := f.httpClient
	if hc == nil {
		hc = http.DefaultClient
	}

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("getting rate limit: %w", forge.ErrNotSupported)
	}
	if resp.StatusCode >= 400 {
		return nil, &forge.HTTPError{StatusCode: resp.StatusCode, URL: url}
	}

	var result giteaRateLimitResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	core := result.Resources.Core
	var reset time.Time
	if core.Reset > 0 {
		reset = time.Unix(core.Reset, 0)
	}

	return &forge.RateLimit{
		Limit:     core.Limit,
		Remaining: core.Remaining,
		Reset:     reset,
	}, nil
}

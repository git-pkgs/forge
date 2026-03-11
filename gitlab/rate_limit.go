package gitlab

import (
	"context"
	"strconv"
	"time"

	forge "github.com/git-pkgs/forge"
)

func (f *gitLabForge) GetRateLimit(ctx context.Context) (*forge.RateLimit, error) {
	// GitLab has no dedicated rate limit endpoint. Rate limit info comes
	// from response headers on any API call, so we make a lightweight request.
	_, resp, err := f.client.Version.GetVersion()
	if err != nil {
		return nil, err
	}

	limit, _ := strconv.Atoi(resp.Header.Get("RateLimit-Limit"))
	remaining, _ := strconv.Atoi(resp.Header.Get("RateLimit-Remaining"))
	resetUnix, _ := strconv.ParseInt(resp.Header.Get("RateLimit-Reset"), 10, 64)

	var reset time.Time
	if resetUnix > 0 {
		reset = time.Unix(resetUnix, 0)
	}

	return &forge.RateLimit{
		Limit:     limit,
		Remaining: remaining,
		Reset:     reset,
	}, nil
}

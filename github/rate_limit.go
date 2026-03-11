package github

import (
	"context"

	forge "github.com/git-pkgs/forge"
)

func (f *gitHubForge) GetRateLimit(ctx context.Context) (*forge.RateLimit, error) {
	limits, _, err := f.client.RateLimit.Get(ctx)
	if err != nil {
		return nil, err
	}

	if limits == nil || limits.Core == nil {
		return &forge.RateLimit{}, nil
	}

	core := limits.Core
	return &forge.RateLimit{
		Limit:     core.Limit,
		Remaining: core.Remaining,
		Reset:     core.Reset.Time,
	}, nil
}

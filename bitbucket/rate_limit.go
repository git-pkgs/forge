package bitbucket

import (
	"context"
	"fmt"

	forge "github.com/git-pkgs/forge"
)

func (f *bitbucketForge) GetRateLimit(ctx context.Context) (*forge.RateLimit, error) {
	return nil, fmt.Errorf("getting rate limit: %w", forge.ErrNotSupported)
}

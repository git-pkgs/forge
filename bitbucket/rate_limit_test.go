package bitbucket

import (
	"context"
	"errors"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestBitbucketRateLimitNotSupported(t *testing.T) {
	f := New("test-token", nil)
	_, err := f.GetRateLimit(context.Background())
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}
}

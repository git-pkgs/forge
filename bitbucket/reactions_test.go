package bitbucket

import (
	"context"
	"errors"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestBitbucketReactionsNotSupported(t *testing.T) {
	f := New("test-token", nil)

	_, err := f.Issues().ListReactions(context.Background(), "owner", "repo", 1, 42)
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}

	_, err = f.Issues().AddReaction(context.Background(), "owner", "repo", 1, 42, "+1")
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}

	_, err = f.PullRequests().ListReactions(context.Background(), "owner", "repo", 1, 42)
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}

	_, err = f.PullRequests().AddReaction(context.Background(), "owner", "repo", 1, 42, "+1")
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}
}

package bitbucket

import (
	"context"
	"fmt"

	forge "github.com/git-pkgs/forge"
)

func (s *bitbucketIssueService) ListReactions(ctx context.Context, owner, repo string, number int, commentID int64) ([]forge.Reaction, error) {
	return nil, fmt.Errorf("listing reactions: %w", forge.ErrNotSupported)
}

func (s *bitbucketIssueService) AddReaction(ctx context.Context, owner, repo string, number int, commentID int64, reaction string) (*forge.Reaction, error) {
	return nil, fmt.Errorf("adding reaction: %w", forge.ErrNotSupported)
}

func (s *bitbucketPRService) ListReactions(ctx context.Context, owner, repo string, number int, commentID int64) ([]forge.Reaction, error) {
	return nil, fmt.Errorf("listing reactions: %w", forge.ErrNotSupported)
}

func (s *bitbucketPRService) AddReaction(ctx context.Context, owner, repo string, number int, commentID int64, reaction string) (*forge.Reaction, error) {
	return nil, fmt.Errorf("adding reaction: %w", forge.ErrNotSupported)
}

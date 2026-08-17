package bitbucket

import (
	"context"

	forge "github.com/git-pkgs/forge"
)

type bitbucketCommitService struct{}

func (f *bitbucketForge) Commits() forge.CommitService {
	return &bitbucketCommitService{}
}

func (s *bitbucketCommitService) ResolveCommit(_ context.Context, _, _, _ string) (string, error) {
	return "", forge.ErrNotSupported
}

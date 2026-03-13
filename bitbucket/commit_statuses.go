package bitbucket

import (
	"context"

	forge "github.com/git-pkgs/forge"
)

type bitbucketCommitStatusService struct{}

func (f *bitbucketForge) CommitStatuses() forge.CommitStatusService {
	return &bitbucketCommitStatusService{}
}

func (s *bitbucketCommitStatusService) List(_ context.Context, _, _, _ string) ([]forge.CommitStatus, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketCommitStatusService) Set(_ context.Context, _, _, _ string, _ forge.SetCommitStatusOpts) (*forge.CommitStatus, error) {
	return nil, forge.ErrNotSupported
}

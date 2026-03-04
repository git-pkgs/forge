package bitbucket

import (
	"context"

	forge "github.com/git-pkgs/forge"
)

type bitbucketBranchService struct{}

func (f *bitbucketForge) Branches() forge.BranchService {
	return &bitbucketBranchService{}
}

func (s *bitbucketBranchService) List(_ context.Context, _, _ string, _ forge.ListBranchOpts) ([]forge.Branch, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketBranchService) Create(_ context.Context, _, _, _, _ string) (*forge.Branch, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketBranchService) Delete(_ context.Context, _, _, _ string) error {
	return forge.ErrNotSupported
}

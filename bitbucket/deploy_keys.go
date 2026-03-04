package bitbucket

import (
	"context"

	forge "github.com/git-pkgs/forge"
)

type bitbucketDeployKeyService struct{}

func (f *bitbucketForge) DeployKeys() forge.DeployKeyService {
	return &bitbucketDeployKeyService{}
}

func (s *bitbucketDeployKeyService) List(_ context.Context, _, _ string, _ forge.ListDeployKeyOpts) ([]forge.DeployKey, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketDeployKeyService) Get(_ context.Context, _, _ string, _ int64) (*forge.DeployKey, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketDeployKeyService) Create(_ context.Context, _, _ string, _ forge.CreateDeployKeyOpts) (*forge.DeployKey, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketDeployKeyService) Delete(_ context.Context, _, _ string, _ int64) error {
	return forge.ErrNotSupported
}

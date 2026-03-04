package forges

import "context"

type bitbucketDeployKeyService struct{}

func (f *bitbucketForge) DeployKeys() DeployKeyService {
	return &bitbucketDeployKeyService{}
}

func (s *bitbucketDeployKeyService) List(_ context.Context, _, _ string, _ ListDeployKeyOpts) ([]DeployKey, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketDeployKeyService) Get(_ context.Context, _, _ string, _ int64) (*DeployKey, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketDeployKeyService) Create(_ context.Context, _, _ string, _ CreateDeployKeyOpts) (*DeployKey, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketDeployKeyService) Delete(_ context.Context, _, _ string, _ int64) error {
	return ErrNotSupported
}

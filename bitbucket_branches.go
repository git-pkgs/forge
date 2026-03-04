package forges

import "context"

type bitbucketBranchService struct{}

func (f *bitbucketForge) Branches() BranchService {
	return &bitbucketBranchService{}
}

func (s *bitbucketBranchService) List(_ context.Context, _, _ string, _ ListBranchOpts) ([]Branch, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketBranchService) Create(_ context.Context, _, _, _, _ string) (*Branch, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketBranchService) Delete(_ context.Context, _, _, _ string) error {
	return ErrNotSupported
}

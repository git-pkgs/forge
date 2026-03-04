package bitbucket

import (
	"context"

	forge "github.com/git-pkgs/forge"
)

type bitbucketLabelService struct{}

func (f *bitbucketForge) Labels() forge.LabelService {
	return &bitbucketLabelService{}
}

func (s *bitbucketLabelService) List(_ context.Context, _, _ string, _ forge.ListLabelOpts) ([]forge.Label, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketLabelService) Get(_ context.Context, _, _, _ string) (*forge.Label, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketLabelService) Create(_ context.Context, _, _ string, _ forge.CreateLabelOpts) (*forge.Label, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketLabelService) Update(_ context.Context, _, _, _ string, _ forge.UpdateLabelOpts) (*forge.Label, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketLabelService) Delete(_ context.Context, _, _, _ string) error {
	return forge.ErrNotSupported
}

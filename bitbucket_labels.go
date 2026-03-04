package forges

import "context"

type bitbucketLabelService struct{}

func (f *bitbucketForge) Labels() LabelService {
	return &bitbucketLabelService{}
}

func (s *bitbucketLabelService) List(_ context.Context, _, _ string, _ ListLabelOpts) ([]Label, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketLabelService) Get(_ context.Context, _, _, _ string) (*Label, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketLabelService) Create(_ context.Context, _, _ string, _ CreateLabelOpts) (*Label, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketLabelService) Update(_ context.Context, _, _, _ string, _ UpdateLabelOpts) (*Label, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketLabelService) Delete(_ context.Context, _, _, _ string) error {
	return ErrNotSupported
}

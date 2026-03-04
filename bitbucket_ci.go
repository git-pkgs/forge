package forges

import (
	"context"
	"io"
)

type bitbucketCIService struct{}

func (f *bitbucketForge) CI() CIService {
	return &bitbucketCIService{}
}

func (s *bitbucketCIService) ListRuns(_ context.Context, _, _ string, _ ListCIRunOpts) ([]CIRun, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketCIService) GetRun(_ context.Context, _, _ string, _ int64) (*CIRun, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketCIService) TriggerRun(_ context.Context, _, _ string, _ TriggerCIRunOpts) error {
	return ErrNotSupported
}

func (s *bitbucketCIService) CancelRun(_ context.Context, _, _ string, _ int64) error {
	return ErrNotSupported
}

func (s *bitbucketCIService) RetryRun(_ context.Context, _, _ string, _ int64) error {
	return ErrNotSupported
}

func (s *bitbucketCIService) GetJobLog(_ context.Context, _, _ string, _ int64) (io.ReadCloser, error) {
	return nil, ErrNotSupported
}

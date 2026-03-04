package bitbucket

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"io"
)

type bitbucketCIService struct{}

func (f *bitbucketForge) CI() forge.CIService {
	return &bitbucketCIService{}
}

func (s *bitbucketCIService) ListRuns(_ context.Context, _, _ string, _ forge.ListCIRunOpts) ([]forge.CIRun, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketCIService) GetRun(_ context.Context, _, _ string, _ int64) (*forge.CIRun, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketCIService) TriggerRun(_ context.Context, _, _ string, _ forge.TriggerCIRunOpts) error {
	return forge.ErrNotSupported
}

func (s *bitbucketCIService) CancelRun(_ context.Context, _, _ string, _ int64) error {
	return forge.ErrNotSupported
}

func (s *bitbucketCIService) RetryRun(_ context.Context, _, _ string, _ int64) error {
	return forge.ErrNotSupported
}

func (s *bitbucketCIService) GetJobLog(_ context.Context, _, _ string, _ int64) (io.ReadCloser, error) {
	return nil, forge.ErrNotSupported
}

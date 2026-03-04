package forges

import (
	"context"
	"io"
)

type giteaCIService struct{}

func (f *giteaForge) CI() CIService {
	return &giteaCIService{}
}

func (s *giteaCIService) ListRuns(_ context.Context, _, _ string, _ ListCIRunOpts) ([]CIRun, error) {
	return nil, ErrNotSupported
}

func (s *giteaCIService) GetRun(_ context.Context, _, _ string, _ int64) (*CIRun, error) {
	return nil, ErrNotSupported
}

func (s *giteaCIService) TriggerRun(_ context.Context, _, _ string, _ TriggerCIRunOpts) error {
	return ErrNotSupported
}

func (s *giteaCIService) CancelRun(_ context.Context, _, _ string, _ int64) error {
	return ErrNotSupported
}

func (s *giteaCIService) RetryRun(_ context.Context, _, _ string, _ int64) error {
	return ErrNotSupported
}

func (s *giteaCIService) GetJobLog(_ context.Context, _, _ string, _ int64) (io.ReadCloser, error) {
	return nil, ErrNotSupported
}

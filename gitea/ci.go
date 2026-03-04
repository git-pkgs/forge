package gitea

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"io"
)

type giteaCIService struct{}

func (f *giteaForge) CI() forge.CIService {
	return &giteaCIService{}
}

func (s *giteaCIService) ListRuns(_ context.Context, _, _ string, _ forge.ListCIRunOpts) ([]forge.CIRun, error) {
	return nil, forge.ErrNotSupported
}

func (s *giteaCIService) GetRun(_ context.Context, _, _ string, _ int64) (*forge.CIRun, error) {
	return nil, forge.ErrNotSupported
}

func (s *giteaCIService) TriggerRun(_ context.Context, _, _ string, _ forge.TriggerCIRunOpts) error {
	return forge.ErrNotSupported
}

func (s *giteaCIService) CancelRun(_ context.Context, _, _ string, _ int64) error {
	return forge.ErrNotSupported
}

func (s *giteaCIService) RetryRun(_ context.Context, _, _ string, _ int64) error {
	return forge.ErrNotSupported
}

func (s *giteaCIService) GetJobLog(_ context.Context, _, _ string, _ int64) (io.ReadCloser, error) {
	return nil, forge.ErrNotSupported
}

package bitbucket

import (
	"context"

	forge "github.com/git-pkgs/forge"
)

type bitbucketMilestoneService struct{}

func (f *bitbucketForge) Milestones() forge.MilestoneService {
	return &bitbucketMilestoneService{}
}

func (s *bitbucketMilestoneService) List(_ context.Context, _, _ string, _ forge.ListMilestoneOpts) ([]forge.Milestone, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketMilestoneService) Get(_ context.Context, _, _ string, _ int) (*forge.Milestone, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketMilestoneService) Create(_ context.Context, _, _ string, _ forge.CreateMilestoneOpts) (*forge.Milestone, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketMilestoneService) Update(_ context.Context, _, _ string, _ int, _ forge.UpdateMilestoneOpts) (*forge.Milestone, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketMilestoneService) Close(_ context.Context, _, _ string, _ int) error {
	return forge.ErrNotSupported
}

func (s *bitbucketMilestoneService) Reopen(_ context.Context, _, _ string, _ int) error {
	return forge.ErrNotSupported
}

func (s *bitbucketMilestoneService) Delete(_ context.Context, _, _ string, _ int) error {
	return forge.ErrNotSupported
}

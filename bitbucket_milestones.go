package forges

import "context"

type bitbucketMilestoneService struct{}

func (f *bitbucketForge) Milestones() MilestoneService {
	return &bitbucketMilestoneService{}
}

func (s *bitbucketMilestoneService) List(_ context.Context, _, _ string, _ ListMilestoneOpts) ([]Milestone, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketMilestoneService) Get(_ context.Context, _, _ string, _ int) (*Milestone, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketMilestoneService) Create(_ context.Context, _, _ string, _ CreateMilestoneOpts) (*Milestone, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketMilestoneService) Update(_ context.Context, _, _ string, _ int, _ UpdateMilestoneOpts) (*Milestone, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketMilestoneService) Close(_ context.Context, _, _ string, _ int) error {
	return ErrNotSupported
}

func (s *bitbucketMilestoneService) Reopen(_ context.Context, _, _ string, _ int) error {
	return ErrNotSupported
}

func (s *bitbucketMilestoneService) Delete(_ context.Context, _, _ string, _ int) error {
	return ErrNotSupported
}

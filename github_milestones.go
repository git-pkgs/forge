package forges

import (
	"context"
	"net/http"

	"github.com/google/go-github/v82/github"
)

type gitHubMilestoneService struct {
	client *github.Client
}

func (f *gitHubForge) Milestones() MilestoneService {
	return &gitHubMilestoneService{client: f.client}
}

func convertGitHubMilestone(m *github.Milestone) Milestone {
	result := Milestone{
		Title:       m.GetTitle(),
		Number:      m.GetNumber(),
		Description: m.GetDescription(),
		State:       m.GetState(),
	}
	if m.DueOn != nil {
		t := m.DueOn.Time
		result.DueDate = &t
	}
	return result
}

func (s *gitHubMilestoneService) List(ctx context.Context, owner, repo string, opts ListMilestoneOpts) ([]Milestone, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.MilestoneListOptions{
		State:       opts.State,
		ListOptions: github.ListOptions{PerPage: perPage, Page: page},
	}

	var all []Milestone
	for {
		milestones, resp, err := s.client.Issues.ListMilestones(ctx, owner, repo, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		for _, m := range milestones {
			all = append(all, convertGitHubMilestone(m))
		}
		if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		ghOpts.ListOptions.Page = resp.NextPage
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *gitHubMilestoneService) Get(ctx context.Context, owner, repo string, id int) (*Milestone, error) {
	m, resp, err := s.client.Issues.GetMilestone(ctx, owner, repo, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubMilestone(m)
	return &result, nil
}

func (s *gitHubMilestoneService) Create(ctx context.Context, owner, repo string, opts CreateMilestoneOpts) (*Milestone, error) {
	ghMilestone := &github.Milestone{
		Title:       &opts.Title,
		Description: &opts.Description,
	}
	if opts.DueDate != nil {
		ghMilestone.DueOn = &github.Timestamp{Time: *opts.DueDate}
	}

	m, resp, err := s.client.Issues.CreateMilestone(ctx, owner, repo, ghMilestone)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubMilestone(m)
	return &result, nil
}

func (s *gitHubMilestoneService) Update(ctx context.Context, owner, repo string, id int, opts UpdateMilestoneOpts) (*Milestone, error) {
	ghMilestone := &github.Milestone{}
	changed := false

	if opts.Title != nil {
		ghMilestone.Title = opts.Title
		changed = true
	}
	if opts.Description != nil {
		ghMilestone.Description = opts.Description
		changed = true
	}
	if opts.State != nil {
		ghMilestone.State = opts.State
		changed = true
	}
	if opts.DueDate != nil {
		ghMilestone.DueOn = &github.Timestamp{Time: *opts.DueDate}
		changed = true
	}

	if !changed {
		return s.Get(ctx, owner, repo, id)
	}

	m, resp, err := s.client.Issues.EditMilestone(ctx, owner, repo, id, ghMilestone)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubMilestone(m)
	return &result, nil
}

func (s *gitHubMilestoneService) Close(ctx context.Context, owner, repo string, id int) error {
	state := "closed"
	ghMilestone := &github.Milestone{State: &state}
	_, resp, err := s.client.Issues.EditMilestone(ctx, owner, repo, id, ghMilestone)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitHubMilestoneService) Reopen(ctx context.Context, owner, repo string, id int) error {
	state := "open"
	ghMilestone := &github.Milestone{State: &state}
	_, resp, err := s.client.Issues.EditMilestone(ctx, owner, repo, id, ghMilestone)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitHubMilestoneService) Delete(ctx context.Context, owner, repo string, id int) error {
	resp, err := s.client.Issues.DeleteMilestone(ctx, owner, repo, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

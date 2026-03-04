package gitlab

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabMilestoneService struct {
	client *gitlab.Client
}

func (f *gitLabForge) Milestones() forge.MilestoneService {
	return &gitLabMilestoneService{client: f.client}
}

func convertGitLabMilestone(m *gitlab.Milestone) forge.Milestone {
	result := forge.Milestone{
		Title:       m.Title,
		Number:      int(m.ID),
		Description: m.Description,
		State:       m.State,
	}
	if m.DueDate != nil {
		t := time.Time(*m.DueDate)
		result.DueDate = &t
	}
	return result
}

func (s *gitLabMilestoneService) List(ctx context.Context, owner, repo string, opts forge.ListMilestoneOpts) ([]forge.Milestone, error) {
	pid := owner + "/" + repo
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	glOpts := &gitlab.ListMilestonesOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage), Page: int64(page)},
	}

	if opts.State != "" && opts.State != "all" {
		glOpts.State = gitlab.Ptr(opts.State)
	}

	var all []forge.Milestone
	for {
		milestones, resp, err := s.client.Milestones.ListMilestones(pid, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, m := range milestones {
			all = append(all, convertGitLabMilestone(m))
		}
		if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		glOpts.Page = int64(resp.NextPage)
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *gitLabMilestoneService) Get(ctx context.Context, owner, repo string, id int) (*forge.Milestone, error) {
	pid := owner + "/" + repo
	m, resp, err := s.client.Milestones.GetMilestone(pid, int64(id))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabMilestone(m)
	return &result, nil
}

func (s *gitLabMilestoneService) Create(ctx context.Context, owner, repo string, opts forge.CreateMilestoneOpts) (*forge.Milestone, error) {
	pid := owner + "/" + repo
	glOpts := &gitlab.CreateMilestoneOptions{
		Title:       gitlab.Ptr(opts.Title),
		Description: gitlab.Ptr(opts.Description),
	}
	if opts.DueDate != nil {
		d := gitlab.ISOTime(*opts.DueDate)
		glOpts.DueDate = &d
	}

	m, resp, err := s.client.Milestones.CreateMilestone(pid, glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabMilestone(m)
	return &result, nil
}

func (s *gitLabMilestoneService) Update(ctx context.Context, owner, repo string, id int, opts forge.UpdateMilestoneOpts) (*forge.Milestone, error) {
	pid := owner + "/" + repo
	glOpts := &gitlab.UpdateMilestoneOptions{}
	changed := false

	if opts.Title != nil {
		glOpts.Title = opts.Title
		changed = true
	}
	if opts.Description != nil {
		glOpts.Description = opts.Description
		changed = true
	}
	if opts.State != nil {
		// GitLab uses state_event: "close" or "activate"
		switch *opts.State {
		case "closed":
			glOpts.StateEvent = gitlab.Ptr("close")
		case "open":
			glOpts.StateEvent = gitlab.Ptr("activate")
		}
		changed = true
	}
	if opts.DueDate != nil {
		d := gitlab.ISOTime(*opts.DueDate)
		glOpts.DueDate = &d
		changed = true
	}

	if !changed {
		return s.Get(ctx, owner, repo, id)
	}

	m, resp, err := s.client.Milestones.UpdateMilestone(pid, int64(id), glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabMilestone(m)
	return &result, nil
}

func (s *gitLabMilestoneService) Close(ctx context.Context, owner, repo string, id int) error {
	pid := owner + "/" + repo
	glOpts := &gitlab.UpdateMilestoneOptions{
		StateEvent: gitlab.Ptr("close"),
	}
	_, resp, err := s.client.Milestones.UpdateMilestone(pid, int64(id), glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabMilestoneService) Reopen(ctx context.Context, owner, repo string, id int) error {
	pid := owner + "/" + repo
	glOpts := &gitlab.UpdateMilestoneOptions{
		StateEvent: gitlab.Ptr("activate"),
	}
	_, resp, err := s.client.Milestones.UpdateMilestone(pid, int64(id), glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabMilestoneService) Delete(ctx context.Context, owner, repo string, id int) error {
	pid := owner + "/" + repo
	resp, err := s.client.Milestones.DeleteMilestone(pid, int64(id))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

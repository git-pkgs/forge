package gitea

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

const (
	stateOpen   = "open"
	stateClosed = "closed"
	stateAll    = "all"
)

type giteaMilestoneService struct {
	client *gitea.Client
}

func (f *giteaForge) Milestones() forge.MilestoneService {
	return &giteaMilestoneService{client: f.client}
}

func convertGiteaMilestone(m *gitea.Milestone) forge.Milestone {
	result := forge.Milestone{
		Title:       m.Title,
		Number:      int(m.ID),
		Description: m.Description,
		State:       string(m.State),
	}
	if m.Deadline != nil && !m.Deadline.IsZero() {
		t := *m.Deadline
		result.DueDate = &t
	}
	return result
}

func (s *giteaMilestoneService) List(ctx context.Context, owner, repo string, opts forge.ListMilestoneOpts) ([]forge.Milestone, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	gOpts := gitea.ListMilestoneOption{
		ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
	}

	switch opts.State {
	case stateOpen:
		gOpts.State = gitea.StateOpen
	case stateClosed:
		gOpts.State = gitea.StateClosed
	case stateAll:
		gOpts.State = gitea.StateAll
	default:
		gOpts.State = gitea.StateOpen
	}

	var all []forge.Milestone
	for {
		milestones, resp, err := s.client.ListRepoMilestones(owner, repo, gOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, m := range milestones {
			all = append(all, convertGiteaMilestone(m))
		}
		if len(milestones) < perPage || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		gOpts.Page++
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *giteaMilestoneService) Get(ctx context.Context, owner, repo string, id int) (*forge.Milestone, error) {
	m, resp, err := s.client.GetMilestone(owner, repo, int64(id))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaMilestone(m)
	return &result, nil
}

func (s *giteaMilestoneService) Create(ctx context.Context, owner, repo string, opts forge.CreateMilestoneOpts) (*forge.Milestone, error) {
	gOpts := gitea.CreateMilestoneOption{
		Title:       opts.Title,
		Description: opts.Description,
	}
	if opts.DueDate != nil {
		gOpts.Deadline = opts.DueDate
	}

	m, resp, err := s.client.CreateMilestone(owner, repo, gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaMilestone(m)
	return &result, nil
}

func (s *giteaMilestoneService) Update(ctx context.Context, owner, repo string, id int, opts forge.UpdateMilestoneOpts) (*forge.Milestone, error) {
	gOpts := gitea.EditMilestoneOption{}
	changed := false

	if opts.Title != nil {
		gOpts.Title = *opts.Title
		changed = true
	}
	if opts.Description != nil {
		gOpts.Description = opts.Description
		changed = true
	}
	if opts.State != nil {
		switch *opts.State {
		case stateOpen:
			s := gitea.StateOpen
			gOpts.State = &s
		case stateClosed:
			s := gitea.StateClosed
			gOpts.State = &s
		}
		changed = true
	}
	if opts.DueDate != nil {
		gOpts.Deadline = opts.DueDate
		changed = true
	}

	if !changed {
		return s.Get(ctx, owner, repo, id)
	}

	m, resp, err := s.client.EditMilestone(owner, repo, int64(id), gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaMilestone(m)
	return &result, nil
}

func (s *giteaMilestoneService) Close(ctx context.Context, owner, repo string, id int) error {
	closed := gitea.StateClosed
	gOpts := gitea.EditMilestoneOption{State: &closed}
	_, resp, err := s.client.EditMilestone(owner, repo, int64(id), gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *giteaMilestoneService) Reopen(ctx context.Context, owner, repo string, id int) error {
	open := gitea.StateOpen
	gOpts := gitea.EditMilestoneOption{State: &open}
	_, resp, err := s.client.EditMilestone(owner, repo, int64(id), gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *giteaMilestoneService) Delete(ctx context.Context, owner, repo string, id int) error {
	resp, err := s.client.DeleteMilestone(owner, repo, int64(id))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

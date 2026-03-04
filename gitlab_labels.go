package forges

import (
	"context"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabLabelService struct {
	client *gitlab.Client
}

func (f *gitLabForge) Labels() LabelService {
	return &gitLabLabelService{client: f.client}
}

func convertGitLabLabel(l *gitlab.Label) Label {
	return Label{
		Name:        l.Name,
		Color:       l.Color,
		Description: l.Description,
	}
}

func (s *gitLabLabelService) List(ctx context.Context, owner, repo string, opts ListLabelOpts) ([]Label, error) {
	pid := owner + "/" + repo
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 100
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	glOpts := &gitlab.ListLabelsOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage), Page: int64(page)},
	}

	var all []Label
	for {
		labels, resp, err := s.client.Labels.ListLabels(pid, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		for _, l := range labels {
			all = append(all, convertGitLabLabel(l))
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

func (s *gitLabLabelService) Get(ctx context.Context, owner, repo, name string) (*Label, error) {
	pid := owner + "/" + repo
	l, resp, err := s.client.Labels.GetLabel(pid, name)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabLabel(l)
	return &result, nil
}

func (s *gitLabLabelService) Create(ctx context.Context, owner, repo string, opts CreateLabelOpts) (*Label, error) {
	pid := owner + "/" + repo
	glOpts := &gitlab.CreateLabelOptions{
		Name:        gitlab.Ptr(opts.Name),
		Color:       gitlab.Ptr(opts.Color),
		Description: gitlab.Ptr(opts.Description),
	}

	l, resp, err := s.client.Labels.CreateLabel(pid, glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabLabel(l)
	return &result, nil
}

func (s *gitLabLabelService) Update(ctx context.Context, owner, repo, name string, opts UpdateLabelOpts) (*Label, error) {
	pid := owner + "/" + repo
	glOpts := &gitlab.UpdateLabelOptions{}
	changed := false

	if opts.Name != nil {
		glOpts.NewName = opts.Name
		changed = true
	}
	if opts.Color != nil {
		glOpts.Color = opts.Color
		changed = true
	}
	if opts.Description != nil {
		glOpts.Description = opts.Description
		changed = true
	}

	if !changed {
		return s.Get(ctx, owner, repo, name)
	}

	l, resp, err := s.client.Labels.UpdateLabel(pid, name, glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabLabel(l)
	return &result, nil
}

func (s *gitLabLabelService) Delete(ctx context.Context, owner, repo, name string) error {
	pid := owner + "/" + repo
	resp, err := s.client.Labels.DeleteLabel(pid, name, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

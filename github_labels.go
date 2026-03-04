package forges

import (
	"context"
	"net/http"

	"github.com/google/go-github/v82/github"
)

type gitHubLabelService struct {
	client *github.Client
}

func (f *gitHubForge) Labels() LabelService {
	return &gitHubLabelService{client: f.client}
}

func convertGitHubLabel(l *github.Label) Label {
	return Label{
		Name:        l.GetName(),
		Color:       l.GetColor(),
		Description: l.GetDescription(),
	}
}

func (s *gitHubLabelService) List(ctx context.Context, owner, repo string, opts ListLabelOpts) ([]Label, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 100
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.ListOptions{PerPage: perPage, Page: page}

	var all []Label
	for {
		labels, resp, err := s.client.Issues.ListLabels(ctx, owner, repo, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		for _, l := range labels {
			all = append(all, convertGitHubLabel(l))
		}
		if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		ghOpts.Page = resp.NextPage
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *gitHubLabelService) Get(ctx context.Context, owner, repo, name string) (*Label, error) {
	l, resp, err := s.client.Issues.GetLabel(ctx, owner, repo, name)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubLabel(l)
	return &result, nil
}

func (s *gitHubLabelService) Create(ctx context.Context, owner, repo string, opts CreateLabelOpts) (*Label, error) {
	ghLabel := &github.Label{
		Name:        &opts.Name,
		Color:       &opts.Color,
		Description: &opts.Description,
	}

	l, resp, err := s.client.Issues.CreateLabel(ctx, owner, repo, ghLabel)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubLabel(l)
	return &result, nil
}

func (s *gitHubLabelService) Update(ctx context.Context, owner, repo, name string, opts UpdateLabelOpts) (*Label, error) {
	ghLabel := &github.Label{}
	changed := false

	if opts.Name != nil {
		ghLabel.Name = opts.Name
		changed = true
	}
	if opts.Color != nil {
		ghLabel.Color = opts.Color
		changed = true
	}
	if opts.Description != nil {
		ghLabel.Description = opts.Description
		changed = true
	}

	if !changed {
		return s.Get(ctx, owner, repo, name)
	}

	l, resp, err := s.client.Issues.EditLabel(ctx, owner, repo, name, ghLabel)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubLabel(l)
	return &result, nil
}

func (s *gitHubLabelService) Delete(ctx context.Context, owner, repo, name string) error {
	resp, err := s.client.Issues.DeleteLabel(ctx, owner, repo, name)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

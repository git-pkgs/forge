package gitea

import (
	"context"
	"fmt"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"strings"

	"code.gitea.io/sdk/gitea"
)

type giteaLabelService struct {
	client *gitea.Client
}

func (f *giteaForge) Labels() forge.LabelService {
	return &giteaLabelService{client: f.client}
}

func convertGiteaLabel(l *gitea.Label) forge.Label {
	return forge.Label{
		Name:        l.Name,
		Color:       l.Color,
		Description: l.Description,
	}
}

func (s *giteaLabelService) List(ctx context.Context, owner, repo string, opts forge.ListLabelOpts) ([]forge.Label, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = defaultPageSize
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	var all []forge.Label
	for {
		labels, resp, err := s.client.ListRepoLabels(owner, repo, gitea.ListLabelsOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, l := range labels {
			all = append(all, convertGiteaLabel(l))
		}
		if len(labels) < perPage || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		page++
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

// findLabelByName lists labels and returns the one matching name.
// Gitea's API uses int64 IDs, so Get/Edit/Delete by name require a lookup.
func (s *giteaLabelService) findLabelByName(owner, repo, name string) (*gitea.Label, error) {
	page := 1
	for {
		labels, resp, err := s.client.ListRepoLabels(owner, repo, gitea.ListLabelsOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: defaultPageSize},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, l := range labels {
			if l.Name == name {
				return l, nil
			}
		}
		if len(labels) < defaultPageSize {
			break
		}
		page++
	}
	return nil, forge.ErrNotFound
}

// resolveLabelIDs maps label names to their numeric IDs by listing
// all labels on the repo. Gitea's API requires IDs for issue/PR creation.
func resolveLabelIDs(client *gitea.Client, owner, repo string, names []string) ([]int64, error) {
	nameSet := make(map[string]struct{}, len(names))
	for _, n := range names {
		nameSet[n] = struct{}{}
	}

	ids := make([]int64, 0, len(names))
	page := 1
	for {
		labels, resp, err := client.ListRepoLabels(owner, repo, gitea.ListLabelsOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: defaultPageSize},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, l := range labels {
			if _, ok := nameSet[l.Name]; ok {
				ids = append(ids, l.ID)
				delete(nameSet, l.Name)
			}
		}
		if len(nameSet) == 0 || len(labels) < defaultPageSize {
			break
		}
		page++
	}

	if len(nameSet) > 0 {
		missing := make([]string, 0, len(nameSet))
		for n := range nameSet {
			missing = append(missing, n)
		}
		return nil, fmt.Errorf("labels not found: %s", strings.Join(missing, ", "))
	}

	return ids, nil
}

func (s *giteaLabelService) Get(ctx context.Context, owner, repo, name string) (*forge.Label, error) {
	l, err := s.findLabelByName(owner, repo, name)
	if err != nil {
		return nil, err
	}
	result := convertGiteaLabel(l)
	return &result, nil
}

func (s *giteaLabelService) Create(ctx context.Context, owner, repo string, opts forge.CreateLabelOpts) (*forge.Label, error) {
	gOpts := gitea.CreateLabelOption{
		Name:        opts.Name,
		Color:       opts.Color,
		Description: opts.Description,
	}

	l, resp, err := s.client.CreateLabel(owner, repo, gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaLabel(l)
	return &result, nil
}

func (s *giteaLabelService) Update(ctx context.Context, owner, repo, name string, opts forge.UpdateLabelOpts) (*forge.Label, error) {
	existing, err := s.findLabelByName(owner, repo, name)
	if err != nil {
		return nil, err
	}

	gOpts := gitea.EditLabelOption{}
	changed := false

	if opts.Name != nil {
		gOpts.Name = opts.Name
		changed = true
	}
	if opts.Color != nil {
		gOpts.Color = opts.Color
		changed = true
	}
	if opts.Description != nil {
		gOpts.Description = opts.Description
		changed = true
	}

	if !changed {
		result := convertGiteaLabel(existing)
		return &result, nil
	}

	l, resp, err := s.client.EditLabel(owner, repo, existing.ID, gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaLabel(l)
	return &result, nil
}

func (s *giteaLabelService) Delete(ctx context.Context, owner, repo, name string) error {
	existing, err := s.findLabelByName(owner, repo, name)
	if err != nil {
		return err
	}

	resp, err := s.client.DeleteLabel(owner, repo, existing.ID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

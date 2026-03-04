package gitea

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

type giteaDeployKeyService struct {
	client *gitea.Client
}

func (f *giteaForge) DeployKeys() forge.DeployKeyService {
	return &giteaDeployKeyService{client: f.client}
}

func (s *giteaDeployKeyService) List(ctx context.Context, owner, repo string, opts forge.ListDeployKeyOpts) ([]forge.DeployKey, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	var all []forge.DeployKey
	for {
		keys, resp, err := s.client.ListDeployKeys(owner, repo, gitea.ListDeployKeysOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, k := range keys {
			all = append(all, forge.DeployKey{
				ID:        k.ID,
				Title:     k.Title,
				Key:       k.Key,
				ReadOnly:  k.ReadOnly,
				CreatedAt: k.Created,
			})
		}
		if len(keys) < perPage || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		page++
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *giteaDeployKeyService) Get(ctx context.Context, owner, repo string, id int64) (*forge.DeployKey, error) {
	k, resp, err := s.client.GetDeployKey(owner, repo, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	return &forge.DeployKey{
		ID:        k.ID,
		Title:     k.Title,
		Key:       k.Key,
		ReadOnly:  k.ReadOnly,
		CreatedAt: k.Created,
	}, nil
}

func (s *giteaDeployKeyService) Create(ctx context.Context, owner, repo string, opts forge.CreateDeployKeyOpts) (*forge.DeployKey, error) {
	k, resp, err := s.client.CreateDeployKey(owner, repo, gitea.CreateKeyOption{
		Title:    opts.Title,
		Key:      opts.Key,
		ReadOnly: opts.ReadOnly,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	return &forge.DeployKey{
		ID:        k.ID,
		Title:     k.Title,
		Key:       k.Key,
		ReadOnly:  k.ReadOnly,
		CreatedAt: k.Created,
	}, nil
}

func (s *giteaDeployKeyService) Delete(ctx context.Context, owner, repo string, id int64) error {
	resp, err := s.client.DeleteDeployKey(owner, repo, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

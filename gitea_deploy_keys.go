package forges

import (
	"context"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

type giteaDeployKeyService struct {
	client *gitea.Client
}

func (f *giteaForge) DeployKeys() DeployKeyService {
	return &giteaDeployKeyService{client: f.client}
}

func (s *giteaDeployKeyService) List(ctx context.Context, owner, repo string, opts ListDeployKeyOpts) ([]DeployKey, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	var all []DeployKey
	for {
		keys, resp, err := s.client.ListDeployKeys(owner, repo, gitea.ListDeployKeysOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		for _, k := range keys {
			all = append(all, DeployKey{
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

func (s *giteaDeployKeyService) Get(ctx context.Context, owner, repo string, id int64) (*DeployKey, error) {
	k, resp, err := s.client.GetDeployKey(owner, repo, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &DeployKey{
		ID:        k.ID,
		Title:     k.Title,
		Key:       k.Key,
		ReadOnly:  k.ReadOnly,
		CreatedAt: k.Created,
	}, nil
}

func (s *giteaDeployKeyService) Create(ctx context.Context, owner, repo string, opts CreateDeployKeyOpts) (*DeployKey, error) {
	k, resp, err := s.client.CreateDeployKey(owner, repo, gitea.CreateKeyOption{
		Title:    opts.Title,
		Key:      opts.Key,
		ReadOnly: opts.ReadOnly,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &DeployKey{
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
			return ErrNotFound
		}
		return err
	}
	return nil
}

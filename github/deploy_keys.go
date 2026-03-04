package github

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"github.com/google/go-github/v82/github"
)

type gitHubDeployKeyService struct {
	client *github.Client
}

func (f *gitHubForge) DeployKeys() forge.DeployKeyService {
	return &gitHubDeployKeyService{client: f.client}
}

func (s *gitHubDeployKeyService) List(ctx context.Context, owner, repo string, opts forge.ListDeployKeyOpts) ([]forge.DeployKey, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.ListOptions{PerPage: perPage, Page: page}

	var all []forge.DeployKey
	for {
		keys, resp, err := s.client.Repositories.ListKeys(ctx, owner, repo, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, k := range keys {
			all = append(all, forge.DeployKey{
				ID:       k.GetID(),
				Title:    k.GetTitle(),
				Key:      k.GetKey(),
				ReadOnly: k.GetReadOnly(),
			})
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

func (s *gitHubDeployKeyService) Get(ctx context.Context, owner, repo string, id int64) (*forge.DeployKey, error) {
	k, resp, err := s.client.Repositories.GetKey(ctx, owner, repo, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	return &forge.DeployKey{
		ID:       k.GetID(),
		Title:    k.GetTitle(),
		Key:      k.GetKey(),
		ReadOnly: k.GetReadOnly(),
	}, nil
}

func (s *gitHubDeployKeyService) Create(ctx context.Context, owner, repo string, opts forge.CreateDeployKeyOpts) (*forge.DeployKey, error) {
	k, resp, err := s.client.Repositories.CreateKey(ctx, owner, repo, &github.Key{
		Title:    github.Ptr(opts.Title),
		Key:      github.Ptr(opts.Key),
		ReadOnly: github.Ptr(opts.ReadOnly),
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	return &forge.DeployKey{
		ID:       k.GetID(),
		Title:    k.GetTitle(),
		Key:      k.GetKey(),
		ReadOnly: k.GetReadOnly(),
	}, nil
}

func (s *gitHubDeployKeyService) Delete(ctx context.Context, owner, repo string, id int64) error {
	resp, err := s.client.Repositories.DeleteKey(ctx, owner, repo, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

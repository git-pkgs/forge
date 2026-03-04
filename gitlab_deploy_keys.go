package forges

import (
	"context"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabDeployKeyService struct {
	client *gitlab.Client
}

func (f *gitLabForge) DeployKeys() DeployKeyService {
	return &gitLabDeployKeyService{client: f.client}
}

func (s *gitLabDeployKeyService) List(ctx context.Context, owner, repo string, opts ListDeployKeyOpts) ([]DeployKey, error) {
	pid := owner + "/" + repo
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	glOpts := &gitlab.ListProjectDeployKeysOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage), Page: int64(page)},
	}

	var all []DeployKey
	for {
		keys, resp, err := s.client.DeployKeys.ListProjectDeployKeys(pid, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		for _, k := range keys {
			dk := DeployKey{
				ID:       k.ID,
				Title:    k.Title,
				Key:      k.Key,
				ReadOnly: !k.CanPush,
			}
			if k.CreatedAt != nil {
				dk.CreatedAt = *k.CreatedAt
			}
			all = append(all, dk)
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

func (s *gitLabDeployKeyService) Get(ctx context.Context, owner, repo string, id int64) (*DeployKey, error) {
	pid := owner + "/" + repo
	k, resp, err := s.client.DeployKeys.GetDeployKey(pid, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	dk := &DeployKey{
		ID:       k.ID,
		Title:    k.Title,
		Key:      k.Key,
		ReadOnly: !k.CanPush,
	}
	if k.CreatedAt != nil {
		dk.CreatedAt = *k.CreatedAt
	}
	return dk, nil
}

func (s *gitLabDeployKeyService) Create(ctx context.Context, owner, repo string, opts CreateDeployKeyOpts) (*DeployKey, error) {
	pid := owner + "/" + repo
	canPush := !opts.ReadOnly
	k, resp, err := s.client.DeployKeys.AddDeployKey(pid, &gitlab.AddDeployKeyOptions{
		Title:   gitlab.Ptr(opts.Title),
		Key:     gitlab.Ptr(opts.Key),
		CanPush: gitlab.Ptr(canPush),
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	dk := &DeployKey{
		ID:       k.ID,
		Title:    k.Title,
		Key:      k.Key,
		ReadOnly: !k.CanPush,
	}
	if k.CreatedAt != nil {
		dk.CreatedAt = *k.CreatedAt
	}
	return dk, nil
}

func (s *gitLabDeployKeyService) Delete(ctx context.Context, owner, repo string, id int64) error {
	pid := owner + "/" + repo
	resp, err := s.client.DeployKeys.DeleteDeployKey(pid, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

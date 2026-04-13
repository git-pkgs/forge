package gitea

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

type giteaSecretService struct {
	client *gitea.Client
}

func (f *giteaForge) Secrets() forge.SecretService {
	return &giteaSecretService{client: f.client}
}

func (s *giteaSecretService) List(ctx context.Context, owner, repo string, opts forge.ListSecretOpts) ([]forge.Secret, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	var all []forge.Secret
	for {
		secrets, resp, err := s.client.ListRepoActionSecret(owner, repo, gitea.ListRepoActionSecretOption{
			ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, sec := range secrets {
			all = append(all, forge.Secret{
				Name:      sec.Name,
				CreatedAt: sec.Created,
			})
		}
		if len(secrets) < perPage || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		page++
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *giteaSecretService) Set(ctx context.Context, owner, repo string, opts forge.SetSecretOpts) error {
	resp, err := s.client.CreateRepoActionSecret(owner, repo, opts.Name, gitea.CreateOrUpdateSecretOption{
		Data: opts.Value,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *giteaSecretService) Delete(ctx context.Context, owner, repo, name string) error {
	resp, err := s.client.DeleteRepoActionSecret(owner, repo, name)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

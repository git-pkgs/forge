package gitlab

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabSecretService struct {
	client *gitlab.Client
}

func (f *gitLabForge) Secrets() forge.SecretService {
	return &gitLabSecretService{client: f.client}
}

func (s *gitLabSecretService) List(ctx context.Context, owner, repo string, opts forge.ListSecretOpts) ([]forge.Secret, error) {
	pid := owner + "/" + repo
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	glOpts := &gitlab.ListProjectVariablesOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage), Page: int64(page)},
	}

	var all []forge.Secret
	for {
		vars, resp, err := s.client.ProjectVariables.ListVariables(pid, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, v := range vars {
			all = append(all, forge.Secret{
				Name: v.Key,
			})
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

func (s *gitLabSecretService) Set(ctx context.Context, owner, repo string, opts forge.SetSecretOpts) error {
	pid := owner + "/" + repo

	// Try update first, fall back to create
	_, resp, err := s.client.ProjectVariables.UpdateVariable(pid, opts.Name, &gitlab.UpdateProjectVariableOptions{
		Value:     gitlab.Ptr(opts.Value),
		Masked:    gitlab.Ptr(true),
		Protected: gitlab.Ptr(false),
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// Variable doesn't exist, create it
			_, _, createErr := s.client.ProjectVariables.CreateVariable(pid, &gitlab.CreateProjectVariableOptions{
				Key:       gitlab.Ptr(opts.Name),
				Value:     gitlab.Ptr(opts.Value),
				Masked:    gitlab.Ptr(true),
				Protected: gitlab.Ptr(false),
			})
			return createErr
		}
		return err
	}
	return nil
}

func (s *gitLabSecretService) Delete(ctx context.Context, owner, repo, name string) error {
	pid := owner + "/" + repo
	resp, err := s.client.ProjectVariables.RemoveVariable(pid, name, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

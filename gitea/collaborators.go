package gitea

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

type giteaCollaboratorService struct {
	client *gitea.Client
}

func (f *giteaForge) Collaborators() forge.CollaboratorService {
	return &giteaCollaboratorService{client: f.client}
}

func (s *giteaCollaboratorService) List(ctx context.Context, owner, repo string, opts forge.ListCollaboratorOpts) ([]forge.Collaborator, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 50
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	var all []forge.Collaborator
	for {
		users, resp, err := s.client.ListCollaborators(owner, repo, gitea.ListCollaboratorsOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, u := range users {
			perm, err := s.getPermission(owner, repo, u.UserName)
			if err != nil {
				perm = "read"
			}
			all = append(all, forge.Collaborator{
				Login:      u.UserName,
				Permission: perm,
			})
		}
		if len(users) < perPage || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		page++
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *giteaCollaboratorService) getPermission(owner, repo, username string) (string, error) {
	result, resp, err := s.client.CollaboratorPermission(owner, repo, username)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return "", forge.ErrNotFound
		}
		return "", err
	}
	switch result.Permission {
	case "admin", "owner":
		return "admin", nil
	case "write":
		return "write", nil
	default:
		return "read", nil
	}
}

func (s *giteaCollaboratorService) Add(ctx context.Context, owner, repo, username string, opts forge.AddCollaboratorOpts) error {
	var perm *gitea.AccessMode
	if opts.Permission != "" {
		mode := gitea.AccessMode(opts.Permission)
		perm = &mode
	}

	resp, err := s.client.AddCollaborator(owner, repo, username, gitea.AddCollaboratorOption{
		Permission: perm,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *giteaCollaboratorService) Remove(ctx context.Context, owner, repo, username string) error {
	resp, err := s.client.DeleteCollaborator(owner, repo, username)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

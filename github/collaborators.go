package github

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"github.com/google/go-github/v82/github"
)

const (
	permRead  = "read"
	permWrite = "write"
	permAdmin = "admin"
)

type gitHubCollaboratorService struct {
	client *github.Client
}

func (f *gitHubForge) Collaborators() forge.CollaboratorService {
	return &gitHubCollaboratorService{client: f.client}
}

func convertGitHubCollaborator(u *github.User) forge.Collaborator {
	perm := permRead
	if u.Permissions != nil {
		if u.Permissions[permAdmin] {
			perm = permAdmin
		} else if u.Permissions["push"] {
			perm = permWrite
		}
	}
	if u.RoleName != nil {
		switch u.GetRoleName() {
		case permAdmin:
			perm = permAdmin
		case permWrite:
			perm = permWrite
		case permRead:
			perm = permRead
		}
	}
	return forge.Collaborator{
		Login:      u.GetLogin(),
		Permission: perm,
	}
}

func (s *gitHubCollaboratorService) List(ctx context.Context, owner, repo string, opts forge.ListCollaboratorOpts) ([]forge.Collaborator, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 100
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.ListCollaboratorsOptions{
		ListOptions: github.ListOptions{PerPage: perPage, Page: page},
	}

	var all []forge.Collaborator
	for {
		users, resp, err := s.client.Repositories.ListCollaborators(ctx, owner, repo, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, u := range users {
			all = append(all, convertGitHubCollaborator(u))
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

func (s *gitHubCollaboratorService) Add(ctx context.Context, owner, repo, username string, opts forge.AddCollaboratorOpts) error {
	ghOpts := &github.RepositoryAddCollaboratorOptions{}
	if opts.Permission != "" {
		ghOpts.Permission = opts.Permission
	}

	_, resp, err := s.client.Repositories.AddCollaborator(ctx, owner, repo, username, ghOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitHubCollaboratorService) Remove(ctx context.Context, owner, repo, username string) error {
	resp, err := s.client.Repositories.RemoveCollaborator(ctx, owner, repo, username)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

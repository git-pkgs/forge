package gitlab

import (
	"context"
	"fmt"
	forge "github.com/git-pkgs/forge"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabCollaboratorService struct {
	client *gitlab.Client
}

func (f *gitLabForge) Collaborators() forge.CollaboratorService {
	return &gitLabCollaboratorService{client: f.client}
}

func convertGitLabAccessLevel(level gitlab.AccessLevelValue) string {
	switch {
	case level >= gitlab.OwnerPermissions:
		return "admin"
	case level >= gitlab.MaintainerPermissions:
		return "admin"
	case level >= gitlab.DeveloperPermissions:
		return "write"
	default:
		return "read"
	}
}

func parseGitLabAccessLevel(permission string) *gitlab.AccessLevelValue {
	var level gitlab.AccessLevelValue
	switch permission {
	case "guest":
		level = gitlab.GuestPermissions
	case "reporter":
		level = gitlab.ReporterPermissions
	case "developer":
		level = gitlab.DeveloperPermissions
	case "maintainer":
		level = gitlab.MaintainerPermissions
	case "owner":
		level = gitlab.OwnerPermissions
	default:
		level = gitlab.DeveloperPermissions
	}
	return &level
}

func convertGitLabMember(m *gitlab.ProjectMember) forge.Collaborator {
	return forge.Collaborator{
		Login:      m.Username,
		Permission: convertGitLabAccessLevel(m.AccessLevel),
	}
}

func (s *gitLabCollaboratorService) List(ctx context.Context, owner, repo string, opts forge.ListCollaboratorOpts) ([]forge.Collaborator, error) {
	pid := owner + "/" + repo
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 100
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	glOpts := &gitlab.ListProjectMembersOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage), Page: int64(page)},
	}

	var all []forge.Collaborator
	for {
		members, resp, err := s.client.ProjectMembers.ListProjectMembers(pid, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, m := range members {
			all = append(all, convertGitLabMember(m))
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

func (s *gitLabCollaboratorService) Add(ctx context.Context, owner, repo, username string, opts forge.AddCollaboratorOpts) error {
	pid := owner + "/" + repo

	// Look up the user ID by username.
	users, resp, err := s.client.Users.ListUsers(&gitlab.ListUsersOptions{
		Username: gitlab.Ptr(username),
	})
	if err != nil {
		return err
	}
	if len(users) == 0 {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return fmt.Errorf("user %q not found", username)
	}

	accessLevel := parseGitLabAccessLevel(opts.Permission)

	_, resp, err = s.client.ProjectMembers.AddProjectMember(pid, &gitlab.AddProjectMemberOptions{
		UserID:      users[0].ID,
		AccessLevel: accessLevel,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabCollaboratorService) Remove(ctx context.Context, owner, repo, username string) error {
	pid := owner + "/" + repo

	// Find the member's user ID by listing members and matching username.
	members, resp, err := s.client.ProjectMembers.ListProjectMembers(pid, &gitlab.ListProjectMembersOptions{})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}

	var userID int64
	found := false
	for _, m := range members {
		if m.Username == username {
			userID = m.ID
			found = true
			break
		}
	}
	if !found {
		return forge.ErrNotFound
	}

	resp, err = s.client.ProjectMembers.DeleteProjectMember(pid, userID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

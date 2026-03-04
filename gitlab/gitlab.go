package gitlab

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabForge struct {
	client *gitlab.Client
}

// New creates a GitLab forge backend.
func New(baseURL, token string, hc *http.Client) forge.Forge {
	opts := []gitlab.ClientOptionFunc{
		gitlab.WithBaseURL(baseURL + "/api/v4"),
	}
	if hc != nil {
		opts = append(opts, gitlab.WithHTTPClient(hc))
	}
	c, _ := gitlab.NewClient(token, opts...)
	return &gitLabForge{client: c}
}

type gitLabRepoService struct {
	client *gitlab.Client
}

func (f *gitLabForge) Repos() forge.RepoService {
	return &gitLabRepoService{client: f.client}
}

func convertGitLabProject(p *gitlab.Project) forge.Repository {
	result := forge.Repository{
		FullName:            p.PathWithNamespace,
		Name:                p.Name,
		Description:         p.Description,
		HTMLURL:             p.WebURL,
		CloneURL:            p.HTTPURLToRepo,
		SSHURL:              p.SSHURLToRepo,
		DefaultBranch:       p.DefaultBranch,
		Archived:            p.Archived,
		Private:             p.Visibility == gitlab.PrivateVisibility,
		StargazersCount:     int(p.StarCount),
		ForksCount:          int(p.ForksCount),
		OpenIssuesCount:     int(p.OpenIssuesCount),
		HasIssues:           true,
		PullRequestsEnabled: p.MergeRequestsAccessLevel != gitlab.DisabledAccessControl,
		Topics:              p.Topics,
	}

	if p.Namespace != nil {
		result.Owner = p.Namespace.Path
		result.LogoURL = p.Namespace.AvatarURL
	}

	if p.License != nil {
		result.License = p.License.Key
	}

	if p.ForkedFromProject != nil {
		result.Fork = true
		result.SourceName = p.ForkedFromProject.PathWithNamespace
	}

	if p.CreatedAt != nil {
		result.CreatedAt = *p.CreatedAt
	}
	if p.LastActivityAt != nil {
		result.UpdatedAt = *p.LastActivityAt
	}

	return result
}

func (s *gitLabRepoService) Get(ctx context.Context, owner, repo string) (*forge.Repository, error) {
	pid := owner + "/" + repo
	license := true
	p, resp, err := s.client.Projects.GetProject(pid, &gitlab.GetProjectOptions{
		License: &license,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	result := convertGitLabProject(p)
	return &result, nil
}

func (s *gitLabRepoService) List(ctx context.Context, owner string, opts forge.ListRepoOpts) ([]forge.Repository, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 100
	}

	// Try group endpoint first, fall back to user projects on 404.
	repos, err := s.listGroupProjects(ctx, owner, perPage)
	if err != nil {
		repos, err = s.listUserProjects(ctx, owner, perPage)
		if err != nil {
			return nil, err
		}
	}

	return forge.FilterRepos(repos, opts), nil
}

func (s *gitLabRepoService) listGroupProjects(ctx context.Context, group string, perPage int) ([]forge.Repository, error) {
	var all []forge.Repository
	glOpts := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage)},
	}
	for {
		projects, resp, err := s.client.Groups.ListGroupProjects(group, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrOwnerNotFound
			}
			return nil, err
		}
		for _, p := range projects {
			all = append(all, convertGitLabProject(p))
		}
		if resp.NextPage == 0 {
			break
		}
		glOpts.Page = resp.NextPage
	}
	return all, nil
}

func (s *gitLabRepoService) listUserProjects(ctx context.Context, user string, perPage int) ([]forge.Repository, error) {
	var all []forge.Repository
	glOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage)},
	}
	for {
		projects, resp, err := s.client.Projects.ListUserProjects(user, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrOwnerNotFound
			}
			return nil, err
		}
		for _, p := range projects {
			all = append(all, convertGitLabProject(p))
		}
		if resp.NextPage == 0 {
			break
		}
		glOpts.Page = resp.NextPage
	}
	return all, nil
}

func (s *gitLabRepoService) Create(ctx context.Context, opts forge.CreateRepoOpts) (*forge.Repository, error) {
	glOpts := &gitlab.CreateProjectOptions{
		Name:        gitlab.Ptr(opts.Name),
		Description: gitlab.Ptr(opts.Description),
	}

	switch opts.Visibility {
	case forge.VisibilityPrivate:
		glOpts.Visibility = gitlab.Ptr(gitlab.PrivateVisibility)
	case forge.VisibilityPublic:
		glOpts.Visibility = gitlab.Ptr(gitlab.PublicVisibility)
	case forge.VisibilityInternal:
		glOpts.Visibility = gitlab.Ptr(gitlab.InternalVisibility)
	}

	if opts.Init || opts.Readme {
		glOpts.InitializeWithReadme = gitlab.Ptr(true)
	}
	if opts.DefaultBranch != "" {
		glOpts.DefaultBranch = gitlab.Ptr(opts.DefaultBranch)
	}

	if opts.Owner != "" {
		// Look up namespace ID for the group
		groups, _, err := s.client.Groups.ListGroups(&gitlab.ListGroupsOptions{
			Search: gitlab.Ptr(opts.Owner),
		})
		if err == nil {
			for _, g := range groups {
				if g.Path == opts.Owner || g.FullPath == opts.Owner {
					glOpts.NamespaceID = gitlab.Ptr(g.ID)
					break
				}
			}
		}
	}

	p, resp, err := s.client.Projects.CreateProject(glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrOwnerNotFound
		}
		return nil, err
	}

	result := convertGitLabProject(p)
	return &result, nil
}

func (s *gitLabRepoService) Edit(ctx context.Context, owner, repo string, opts forge.EditRepoOpts) (*forge.Repository, error) {
	pid := owner + "/" + repo
	glOpts := &gitlab.EditProjectOptions{}
	changed := false

	if opts.Description != nil {
		glOpts.Description = opts.Description
		changed = true
	}
	if opts.DefaultBranch != nil {
		glOpts.DefaultBranch = opts.DefaultBranch
		changed = true
	}

	switch opts.Visibility {
	case forge.VisibilityPrivate:
		glOpts.Visibility = gitlab.Ptr(gitlab.PrivateVisibility)
		changed = true
	case forge.VisibilityPublic:
		glOpts.Visibility = gitlab.Ptr(gitlab.PublicVisibility)
		changed = true
	case forge.VisibilityInternal:
		glOpts.Visibility = gitlab.Ptr(gitlab.InternalVisibility)
		changed = true
	}

	if !changed {
		return s.Get(ctx, owner, repo)
	}

	p, resp, err := s.client.Projects.EditProject(pid, glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	result := convertGitLabProject(p)
	return &result, nil
}

func (s *gitLabRepoService) Delete(ctx context.Context, owner, repo string) error {
	pid := owner + "/" + repo
	resp, err := s.client.Projects.DeleteProject(pid, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabRepoService) Fork(ctx context.Context, owner, repo string, opts forge.ForkRepoOpts) (*forge.Repository, error) {
	pid := owner + "/" + repo
	glOpts := &gitlab.ForkProjectOptions{}
	if opts.Owner != "" {
		glOpts.NamespacePath = gitlab.Ptr(opts.Owner)
	}
	if opts.Name != "" {
		glOpts.Name = gitlab.Ptr(opts.Name)
		glOpts.Path = gitlab.Ptr(opts.Name)
	}

	p, resp, err := s.client.Projects.ForkProject(pid, glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	result := convertGitLabProject(p)
	return &result, nil
}

func (s *gitLabRepoService) ListTags(ctx context.Context, owner, repo string) ([]forge.Tag, error) {
	pid := owner + "/" + repo
	var allTags []forge.Tag
	opts := &gitlab.ListTagsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}
	for {
		tags, resp, err := s.client.Tags.ListTags(pid, opts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, t := range tags {
			tag := forge.Tag{Name: t.Name}
			if t.Commit != nil {
				tag.Commit = t.Commit.ID
			}
			allTags = append(allTags, tag)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allTags, nil
}

func (s *gitLabRepoService) Search(ctx context.Context, opts forge.SearchRepoOpts) ([]forge.Repository, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	glOpts := &gitlab.ListProjectsOptions{
		Search:      gitlab.Ptr(opts.Query),
		OrderBy:     gitlab.Ptr(opts.Sort),
		Sort:        gitlab.Ptr(opts.Order),
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage), Page: int64(page)},
	}

	projects, resp, err := s.client.Projects.ListProjects(glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}

	var repos []forge.Repository
	for _, p := range projects {
		repos = append(repos, convertGitLabProject(p))
	}
	return repos, nil
}

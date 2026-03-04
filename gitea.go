package forges

import (
	"context"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

type giteaForge struct {
	client *gitea.Client
}

func newGiteaForge(baseURL, token string, hc *http.Client) *giteaForge {
	opts := []gitea.ClientOption{}
	if token != "" {
		opts = append(opts, gitea.SetToken(token))
	}
	if hc != nil {
		opts = append(opts, gitea.SetHTTPClient(hc))
	}
	c, _ := gitea.NewClient(baseURL, opts...)
	return &giteaForge{client: c}
}

type giteaRepoService struct {
	client *gitea.Client
}

func (f *giteaForge) Repos() RepoService {
	return &giteaRepoService{client: f.client}
}

func convertGiteaRepo(r *gitea.Repository) Repository {
	result := Repository{
		FullName:            r.FullName,
		Owner:               r.Owner.UserName,
		Name:                r.Name,
		Description:         r.Description,
		Homepage:            r.Website,
		HTMLURL:             r.HTMLURL,
		CloneURL:            r.CloneURL,
		SSHURL:              r.SSHURL,
		Language:            r.Language,
		DefaultBranch:       r.DefaultBranch,
		Fork:                r.Fork,
		Archived:            r.Archived,
		Private:             r.Private,
		Size:                int(r.Size),
		StargazersCount:     r.Stars,
		ForksCount:          r.Forks,
		OpenIssuesCount:     r.OpenIssues,
		HasIssues:           r.HasIssues,
		PullRequestsEnabled: r.HasPullRequests,
		LogoURL:             r.AvatarURL,
		CreatedAt:           r.Created,
		UpdatedAt:           r.Updated,
	}

	if r.Mirror {
		result.MirrorURL = r.OriginalURL
	}

	if r.Parent != nil {
		result.SourceName = r.Parent.FullName
	}

	return result
}

func (s *giteaRepoService) Get(ctx context.Context, owner, repo string) (*Repository, error) {
	r, resp, err := s.client.GetRepo(owner, repo)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	result := convertGiteaRepo(r)

	// Fetch topics separately (not included in main repo response)
	topics, _, topicErr := s.client.ListRepoTopics(owner, repo, gitea.ListRepoTopicsOptions{})
	if topicErr == nil {
		result.Topics = topics
	}

	return &result, nil
}

func (s *giteaRepoService) List(ctx context.Context, owner string, opts ListRepoOpts) ([]Repository, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 50
	}

	// Try org endpoint first, fall back to user on 404.
	repos, err := s.listOrgRepos(ctx, owner, perPage)
	if err != nil {
		repos, err = s.listUserRepos(ctx, owner, perPage)
		if err != nil {
			return nil, err
		}
	}

	return FilterRepos(repos, opts), nil
}

func (s *giteaRepoService) listOrgRepos(_ context.Context, owner string, perPage int) ([]Repository, error) {
	var all []Repository
	page := 1
	for {
		gRepos, resp, err := s.client.ListOrgRepos(owner, gitea.ListOrgReposOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrOwnerNotFound
			}
			return nil, err
		}
		for _, r := range gRepos {
			all = append(all, convertGiteaRepo(r))
		}
		if len(gRepos) < perPage {
			break
		}
		page++
	}
	return all, nil
}

func (s *giteaRepoService) listUserRepos(_ context.Context, owner string, perPage int) ([]Repository, error) {
	var all []Repository
	page := 1
	for {
		gRepos, resp, err := s.client.ListUserRepos(owner, gitea.ListReposOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrOwnerNotFound
			}
			return nil, err
		}
		for _, r := range gRepos {
			all = append(all, convertGiteaRepo(r))
		}
		if len(gRepos) < perPage {
			break
		}
		page++
	}
	return all, nil
}

func (s *giteaRepoService) Create(ctx context.Context, opts CreateRepoOpts) (*Repository, error) {
	gOpts := gitea.CreateRepoOption{
		Name:        opts.Name,
		Description: opts.Description,
		AutoInit:    opts.Init || opts.Readme,
	}

	switch opts.Visibility {
	case VisibilityPrivate:
		gOpts.Private = true
	case VisibilityPublic:
		gOpts.Private = false
	}

	if opts.DefaultBranch != "" {
		gOpts.DefaultBranch = opts.DefaultBranch
	}
	if opts.Gitignore != "" {
		gOpts.Gitignores = opts.Gitignore
	}
	if opts.License != "" {
		gOpts.License = opts.License
	}
	if opts.Readme {
		gOpts.Readme = "Default"
	}

	var (
		r    *gitea.Repository
		resp *gitea.Response
		err  error
	)

	if opts.Owner != "" {
		r, resp, err = s.client.CreateOrgRepo(opts.Owner, gOpts)
	} else {
		r, resp, err = s.client.CreateRepo(gOpts)
	}

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrOwnerNotFound
		}
		return nil, err
	}

	result := convertGiteaRepo(r)
	return &result, nil
}

func (s *giteaRepoService) Edit(ctx context.Context, owner, repo string, opts EditRepoOpts) (*Repository, error) {
	gOpts := gitea.EditRepoOption{}
	changed := false

	if opts.Description != nil {
		gOpts.Description = opts.Description
		changed = true
	}
	if opts.Homepage != nil {
		gOpts.Website = opts.Homepage
		changed = true
	}
	if opts.DefaultBranch != nil {
		gOpts.DefaultBranch = opts.DefaultBranch
		changed = true
	}
	if opts.HasIssues != nil {
		gOpts.HasIssues = opts.HasIssues
		changed = true
	}
	if opts.HasPRs != nil {
		gOpts.HasPullRequests = opts.HasPRs
		changed = true
	}

	switch opts.Visibility {
	case VisibilityPrivate:
		gOpts.Private = boolPtr(true)
		changed = true
	case VisibilityPublic:
		gOpts.Private = boolPtr(false)
		changed = true
	}

	if !changed {
		return s.Get(ctx, owner, repo)
	}

	r, resp, err := s.client.EditRepo(owner, repo, gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	result := convertGiteaRepo(r)
	return &result, nil
}

func (s *giteaRepoService) Delete(ctx context.Context, owner, repo string) error {
	resp, err := s.client.DeleteRepo(owner, repo)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *giteaRepoService) Fork(ctx context.Context, owner, repo string, opts ForkRepoOpts) (*Repository, error) {
	gOpts := gitea.CreateForkOption{}
	if opts.Owner != "" {
		o := opts.Owner
		gOpts.Organization = &o
	}
	if opts.Name != "" {
		n := opts.Name
		gOpts.Name = &n
	}

	r, resp, err := s.client.CreateFork(owner, repo, gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	result := convertGiteaRepo(r)
	return &result, nil
}

func (s *giteaRepoService) ListTags(ctx context.Context, owner, repo string) ([]Tag, error) {
	var allTags []Tag
	page := 1
	for {
		tags, resp, err := s.client.ListRepoTags(owner, repo, gitea.ListRepoTagsOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: 50},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		for _, t := range tags {
			tag := Tag{Name: t.Name}
			if t.Commit != nil {
				tag.Commit = t.Commit.SHA
			}
			allTags = append(allTags, tag)
		}
		if len(tags) < 50 {
			break
		}
		page++
	}
	return allTags, nil
}

func (s *giteaRepoService) Search(ctx context.Context, opts SearchRepoOpts) ([]Repository, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	gOpts := gitea.SearchRepoOptions{
		Keyword:     opts.Query,
		ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
	}

	switch opts.Sort {
	case "stars":
		gOpts.Sort = "stars"
	case "forks":
		gOpts.Sort = "forks"
	case "updated":
		gOpts.Sort = "updated"
	default:
		gOpts.Sort = "relevance"
	}

	results, resp, err := s.client.SearchRepos(gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}

	var repos []Repository
	for _, r := range results {
		repos = append(repos, convertGiteaRepo(r))
	}
	return repos, nil
}

func boolPtr(b bool) *bool { return &b }

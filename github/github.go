package github

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"github.com/google/go-github/v82/github"
)

type gitHubForge struct {
	client *github.Client
}

// New creates a GitHub forge backend for github.com.
func New(token string, hc *http.Client) forge.Forge {
	c := github.NewClient(hc)
	if token != "" {
		c = c.WithAuthToken(token)
	}
	return &gitHubForge{client: c}
}

// NewWithBase creates a GitHub forge backend for a GitHub Enterprise instance.
func NewWithBase(baseURL, token string, hc *http.Client) forge.Forge {
	c := github.NewClient(hc).WithAuthToken(token)
	c, _ = c.WithEnterpriseURLs(baseURL, baseURL)
	return &gitHubForge{client: c}
}

type gitHubRepoService struct {
	client *github.Client
}

func (f *gitHubForge) Repos() forge.RepoService {
	return &gitHubRepoService{client: f.client}
}

func convertGitHubRepo(r *github.Repository) forge.Repository {
	result := forge.Repository{
		FullName:            r.GetFullName(),
		Owner:               r.GetOwner().GetLogin(),
		Name:                r.GetName(),
		Description:         r.GetDescription(),
		Homepage:            r.GetHomepage(),
		HTMLURL:             r.GetHTMLURL(),
		CloneURL:            r.GetCloneURL(),
		SSHURL:              r.GetSSHURL(),
		Language:            r.GetLanguage(),
		DefaultBranch:       r.GetDefaultBranch(),
		Fork:                r.GetFork(),
		Archived:            r.GetArchived(),
		Private:             r.GetPrivate(),
		MirrorURL:           r.GetMirrorURL(),
		Size:                r.GetSize(),
		StargazersCount:     r.GetStargazersCount(),
		ForksCount:          r.GetForksCount(),
		OpenIssuesCount:     r.GetOpenIssuesCount(),
		SubscribersCount:    r.GetSubscribersCount(),
		HasIssues:           r.GetHasIssues(),
		PullRequestsEnabled: true, // GitHub always has PRs enabled
		Topics:              r.Topics,
		LogoURL:             r.GetOwner().GetAvatarURL(),
	}

	if lic := r.GetLicense(); lic != nil {
		spdx := lic.GetSPDXID()
		if spdx != "" && spdx != "NOASSERTION" {
			result.License = spdx
		}
	}

	if parent := r.GetParent(); parent != nil {
		result.SourceName = parent.GetFullName()
	}

	if t := r.GetCreatedAt(); !t.IsZero() {
		result.CreatedAt = t.Time
	}
	if t := r.GetUpdatedAt(); !t.IsZero() {
		result.UpdatedAt = t.Time
	}
	if t := r.GetPushedAt(); !t.IsZero() {
		result.PushedAt = t.Time
	}

	return result
}

func (s *gitHubRepoService) Get(ctx context.Context, owner, repo string) (*forge.Repository, error) {
	r, resp, err := s.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	result := convertGitHubRepo(r)
	return &result, nil
}

func (s *gitHubRepoService) List(ctx context.Context, owner string, opts forge.ListRepoOpts) ([]forge.Repository, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 100
	}

	// Try org endpoint first, fall back to user on 404.
	repos, err := s.listOrgRepos(ctx, owner, perPage)
	if err != nil {
		repos, err = s.listUserRepos(ctx, owner, perPage)
		if err != nil {
			return nil, err
		}
	}

	return forge.FilterRepos(repos, opts), nil
}

func (s *gitHubRepoService) listOrgRepos(ctx context.Context, owner string, perPage int) ([]forge.Repository, error) {
	var all []forge.Repository
	ghOpts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: perPage},
	}
	for {
		ghRepos, resp, err := s.client.Repositories.ListByOrg(ctx, owner, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrOwnerNotFound
			}
			return nil, err
		}
		for _, r := range ghRepos {
			all = append(all, convertGitHubRepo(r))
		}
		if resp.NextPage == 0 {
			break
		}
		ghOpts.Page = resp.NextPage
	}
	return all, nil
}

func (s *gitHubRepoService) listUserRepos(ctx context.Context, owner string, perPage int) ([]forge.Repository, error) {
	var all []forge.Repository
	ghOpts := &github.RepositoryListByUserOptions{
		ListOptions: github.ListOptions{PerPage: perPage},
	}
	for {
		ghRepos, resp, err := s.client.Repositories.ListByUser(ctx, owner, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrOwnerNotFound
			}
			return nil, err
		}
		for _, r := range ghRepos {
			all = append(all, convertGitHubRepo(r))
		}
		if resp.NextPage == 0 {
			break
		}
		ghOpts.Page = resp.NextPage
	}
	return all, nil
}

func (s *gitHubRepoService) Create(ctx context.Context, opts forge.CreateRepoOpts) (*forge.Repository, error) {
	ghRepo := &github.Repository{
		Name:        github.Ptr(opts.Name),
		Description: github.Ptr(opts.Description),
		AutoInit:    github.Ptr(opts.Init || opts.Readme),
	}

	switch opts.Visibility {
	case forge.VisibilityPrivate:
		ghRepo.Private = github.Ptr(true)
	case forge.VisibilityInternal:
		ghRepo.Visibility = github.Ptr("internal")
	case forge.VisibilityPublic:
		ghRepo.Private = github.Ptr(false)
	}

	if opts.Gitignore != "" {
		ghRepo.GitignoreTemplate = github.Ptr(opts.Gitignore)
	}
	if opts.License != "" {
		ghRepo.LicenseTemplate = github.Ptr(opts.License)
	}

	var (
		r    *github.Repository
		resp *github.Response
		err  error
	)

	if opts.Owner != "" {
		r, resp, err = s.client.Repositories.Create(ctx, opts.Owner, ghRepo)
	} else {
		r, resp, err = s.client.Repositories.Create(ctx, "", ghRepo)
	}

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrOwnerNotFound
		}
		return nil, err
	}

	result := convertGitHubRepo(r)
	return &result, nil
}

func (s *gitHubRepoService) Edit(ctx context.Context, owner, repo string, opts forge.EditRepoOpts) (*forge.Repository, error) {
	ghRepo := &github.Repository{}
	changed := false

	if opts.Description != nil {
		ghRepo.Description = opts.Description
		changed = true
	}
	if opts.Homepage != nil {
		ghRepo.Homepage = opts.Homepage
		changed = true
	}
	if opts.DefaultBranch != nil {
		ghRepo.DefaultBranch = opts.DefaultBranch
		changed = true
	}
	if opts.HasIssues != nil {
		ghRepo.HasIssues = opts.HasIssues
		changed = true
	}
	if opts.HasPRs != nil {
		// GitHub doesn't support disabling PRs via API, but we set it anyway
		changed = true
	}

	switch opts.Visibility {
	case forge.VisibilityPrivate:
		ghRepo.Private = github.Ptr(true)
		changed = true
	case forge.VisibilityPublic:
		ghRepo.Private = github.Ptr(false)
		changed = true
	case forge.VisibilityInternal:
		ghRepo.Visibility = github.Ptr("internal")
		changed = true
	}

	if !changed {
		return s.Get(ctx, owner, repo)
	}

	r, resp, err := s.client.Repositories.Edit(ctx, owner, repo, ghRepo)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	result := convertGitHubRepo(r)
	return &result, nil
}

func (s *gitHubRepoService) Delete(ctx context.Context, owner, repo string) error {
	resp, err := s.client.Repositories.Delete(ctx, owner, repo)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitHubRepoService) Fork(ctx context.Context, owner, repo string, opts forge.ForkRepoOpts) (*forge.Repository, error) {
	ghOpts := &github.RepositoryCreateForkOptions{}
	if opts.Owner != "" {
		ghOpts.Organization = opts.Owner
	}
	if opts.Name != "" {
		ghOpts.Name = opts.Name
	}

	r, resp, err := s.client.Repositories.CreateFork(ctx, owner, repo, ghOpts)
	if err != nil {
		// GitHub returns 202 Accepted for forks, go-github may return an AcceptedError
		if _, ok := err.(*github.AcceptedError); ok && r != nil {
			result := convertGitHubRepo(r)
			return &result, nil
		}
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	result := convertGitHubRepo(r)
	return &result, nil
}

func (s *gitHubRepoService) ListForks(ctx context.Context, owner, repo string, opts forge.ListForksOpts) ([]forge.Repository, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 100
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.RepositoryListForksOptions{
		Sort:        opts.Sort,
		ListOptions: github.ListOptions{PerPage: perPage, Page: page},
	}

	var all []forge.Repository
	for {
		forks, resp, err := s.client.Repositories.ListForks(ctx, owner, repo, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, r := range forks {
			all = append(all, convertGitHubRepo(r))
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

func (s *gitHubRepoService) ListTags(ctx context.Context, owner, repo string) ([]forge.Tag, error) {
	var allTags []forge.Tag
	opts := &github.ListOptions{PerPage: 100}
	for {
		tags, resp, err := s.client.Repositories.ListTags(ctx, owner, repo, opts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, t := range tags {
			tag := forge.Tag{Name: t.GetName()}
			if c := t.GetCommit(); c != nil {
				tag.Commit = c.GetSHA()
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

func (s *gitHubRepoService) Search(ctx context.Context, opts forge.SearchRepoOpts) ([]forge.Repository, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.SearchOptions{
		Sort:        opts.Sort,
		Order:       opts.Order,
		ListOptions: github.ListOptions{PerPage: perPage, Page: page},
	}

	result, resp, err := s.client.Search.Repositories(ctx, opts.Query, ghOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}

	var repos []forge.Repository
	for _, r := range result.Repositories {
		repos = append(repos, convertGitHubRepo(r))
	}
	return repos, nil
}

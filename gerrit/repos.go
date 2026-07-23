package gerrit

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	forge "github.com/git-pkgs/forge"
)

type gerritRepoService struct {
	forge *gerritForge
}

type gerritProjectInfo struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Parent       string            `json:"parent"`
	Description  string            `json:"description"`
	State        string            `json:"state"`
	Branches     map[string]string `json:"branches"`
	MoreProjects bool              `json:"_more_projects"`
}

func (f *gerritForge) Repos() forge.RepoService {
	return &gerritRepoService{forge: f}
}

func (s *gerritRepoService) convertProject(p gerritProjectInfo) forge.Repository {
	name := p.Name
	if name == "" {
		name = p.ID
	}
	if unescaped, err := url.PathUnescape(name); err == nil {
		name = unescaped
	}
	owner, repo := splitProjectName(name)

	result := forge.Repository{
		FullName:            name,
		Owner:               owner,
		Name:                repo,
		Description:         p.Description,
		HTMLURL:             s.forge.baseURL + "/admin/repos/" + encodeID(name),
		CloneURL:            s.forge.baseURL + "/" + name,
		DefaultBranch:       trimRefPrefix(p.Branches["HEAD"]),
		Archived:            p.State == "READ_ONLY",
		Private:             p.State == "HIDDEN",
		HasIssues:           false,
		PullRequestsEnabled: true,
	}
	if result.DefaultBranch == "" {
		result.DefaultBranch = trimRefPrefix(p.Branches["master"])
	}
	return result
}

func (s *gerritRepoService) Get(ctx context.Context, owner, repo string) (*forge.Repository, error) {
	project := projectName(owner, repo)
	var p gerritProjectInfo
	if err := s.forge.doJSON(ctx, http.MethodGet, "/projects/"+encodeID(project), nil, nil, &p); err != nil {
		return nil, err
	}
	if p.Name == "" {
		p.Name = project
	}

	var head string
	if err := s.forge.doJSON(ctx, http.MethodGet, "/projects/"+encodeID(project)+"/HEAD", nil, nil, &head); err == nil {
		if p.Branches == nil {
			p.Branches = make(map[string]string)
		}
		p.Branches["HEAD"] = head
	}

	result := s.convertProject(p)
	return &result, nil
}

func (s *gerritRepoService) List(ctx context.Context, owner string, opts forge.ListRepoOpts) ([]forge.Repository, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = defaultPageSize
	}
	start := 0
	if opts.Page > 1 {
		start = (opts.Page - 1) * perPage
	}

	var repos []forge.Repository
	for {
		query := url.Values{}
		query.Set("d", "")
		query.Set("n", strconv.Itoa(perPage))
		query.Set("S", strconv.Itoa(start))
		if owner != "" {
			query.Set("p", owner+"/")
		}

		var page map[string]gerritProjectInfo
		if err := s.forge.doJSON(ctx, http.MethodGet, "/projects/", query, nil, &page); err != nil {
			return nil, err
		}

		more := false
		for _, name := range sortedMapKeys(page) {
			p := page[name]
			if p.MoreProjects {
				more = true
			}
			if p.Name == "" {
				p.Name = name
			}
			repos = append(repos, s.convertProject(p))
			if opts.Limit > 0 && len(repos) >= opts.Limit {
				return forge.FilterRepos(repos[:opts.Limit], opts), nil
			}
		}
		if !more || len(page) == 0 {
			break
		}
		start += perPage
	}

	return forge.FilterRepos(repos, opts), nil
}

func (s *gerritRepoService) Create(ctx context.Context, opts forge.CreateRepoOpts) (*forge.Repository, error) {
	name := opts.Name
	if opts.Owner != "" {
		name = opts.Owner + "/" + opts.Name
	}
	body := map[string]any{}
	if opts.Description != "" {
		body["description"] = opts.Description
	}
	if opts.DefaultBranch != "" {
		body["branches"] = []string{opts.DefaultBranch}
	}

	var p gerritProjectInfo
	if err := s.forge.doJSON(ctx, http.MethodPut, "/projects/"+encodeID(name), nil, body, &p); err != nil {
		return nil, err
	}
	if p.Name == "" {
		p.Name = name
	}
	result := s.convertProject(p)
	return &result, nil
}

func (s *gerritRepoService) Edit(ctx context.Context, owner, repo string, opts forge.EditRepoOpts) (*forge.Repository, error) {
	project := projectName(owner, repo)
	if opts.Description != nil {
		body := map[string]string{"description": *opts.Description}
		if err := s.forge.doJSON(ctx, http.MethodPut, "/projects/"+encodeID(project)+"/description", nil, body, nil); err != nil {
			return nil, err
		}
	}
	if opts.DefaultBranch != nil {
		body := map[string]string{"ref": *opts.DefaultBranch}
		if !strings.HasPrefix(*opts.DefaultBranch, "refs/") {
			body["ref"] = "refs/heads/" + *opts.DefaultBranch
		}
		if err := s.forge.doJSON(ctx, http.MethodPut, "/projects/"+encodeID(project)+"/HEAD", nil, body, nil); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, owner, repo)
}

func (s *gerritRepoService) Delete(_ context.Context, _, _ string) error {
	return forge.ErrNotSupported
}

func (s *gerritRepoService) Fork(_ context.Context, _, _ string, _ forge.ForkRepoOpts) (*forge.Repository, error) {
	return nil, forge.ErrNotSupported
}

func (s *gerritRepoService) ListForks(_ context.Context, _, _ string, _ forge.ListForksOpts) ([]forge.Repository, error) {
	return nil, forge.ErrNotSupported
}

func (s *gerritRepoService) ListTags(ctx context.Context, owner, repo string) ([]forge.Tag, error) {
	project := projectName(owner, repo)
	var infos []struct {
		Ref      string `json:"ref"`
		Revision string `json:"revision"`
	}
	if err := s.forge.doJSON(ctx, http.MethodGet, "/projects/"+encodeID(project)+"/tags/", nil, nil, &infos); err != nil {
		return nil, err
	}

	tags := make([]forge.Tag, len(infos))
	for i, info := range infos {
		tags[i] = forge.Tag{Name: trimRefPrefix(info.Ref), Commit: info.Revision}
	}
	return tags, nil
}

func (s *gerritRepoService) ListContributors(_ context.Context, _, _ string) ([]forge.Contributor, error) {
	return nil, forge.ErrNotSupported
}

func (s *gerritRepoService) Search(ctx context.Context, opts forge.SearchRepoOpts) ([]forge.Repository, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = defaultPageSize
	}
	start := 0
	if opts.Page > 1 {
		start = (opts.Page - 1) * perPage
	}

	var repos []forge.Repository
	for {
		query := url.Values{}
		query.Set("query", opts.Query)
		query.Set("limit", strconv.Itoa(perPage))
		query.Set("start", strconv.Itoa(start))

		var page []gerritProjectInfo
		if err := s.forge.doJSON(ctx, http.MethodGet, "/projects/", query, nil, &page); err != nil {
			return nil, err
		}
		more := false
		for _, p := range page {
			if p.MoreProjects {
				more = true
			}
			repos = append(repos, s.convertProject(p))
			if opts.Limit > 0 && len(repos) >= opts.Limit {
				return repos[:opts.Limit], nil
			}
		}
		if !more || len(page) == 0 {
			break
		}
		start += perPage
	}
	return repos, nil
}

func (s *gerritRepoService) SettingsURL(repoHTMLURL string) string {
	return repoHTMLURL
}

func (s *gerritRepoService) WikiURL(repoHTMLURL string) string {
	return repoHTMLURL
}

func (s *gerritRepoService) ActionsURL(repoHTMLURL string) string {
	return repoHTMLURL
}

func (s *gerritRepoService) ReleasesURL(repoHTMLURL string) string {
	return repoHTMLURL
}

func (s *gerritRepoService) BlobURL(repoHTMLURL, ref, filePath string) string {
	return repoHTMLURL
}

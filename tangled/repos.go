package tangled

import (
	"context"
	"net/url"
	"strings"

	forges "github.com/git-pkgs/forge"
)

type repoService struct {
	f *tangledForge
}

func (s *repoService) Get(ctx context.Context, owner, repo string) (*forges.Repository, error) {
	meta, err := s.f.repoMeta(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	cloneURL := meta.CloneURL
	if cloneURL == "" {
		cloneURL = s.f.repoURL(owner, repo)
	}
	result := &forges.Repository{
		FullName:            owner + "/" + repo,
		Owner:               owner,
		Name:                repo,
		Description:         meta.Description,
		HTMLURL:             s.f.repoURL(owner, repo),
		CloneURL:            cloneURL,
		HasIssues:           true,
		PullRequestsEnabled: true,
	}
	if branches, err := s.f.Branches().List(ctx, owner, repo, forges.ListBranchOpts{Limit: 1}); err == nil && len(branches) > 0 {
		result.DefaultBranch = branches[0].Name
	}
	return result, nil
}

func (s *repoService) List(context.Context, string, forges.ListRepoOpts) ([]forges.Repository, error) {
	return nil, forges.ErrNotSupported
}

func (s *repoService) Create(context.Context, forges.CreateRepoOpts) (*forges.Repository, error) {
	return nil, forges.ErrNotSupported
}

func (s *repoService) Edit(context.Context, string, string, forges.EditRepoOpts) (*forges.Repository, error) {
	return nil, forges.ErrNotSupported
}

func (s *repoService) Delete(context.Context, string, string) error {
	return forges.ErrNotSupported
}

func (s *repoService) Fork(context.Context, string, string, forges.ForkRepoOpts) (*forges.Repository, error) {
	return nil, forges.ErrNotSupported
}

func (s *repoService) ListForks(context.Context, string, string, forges.ListForksOpts) ([]forges.Repository, error) {
	return nil, forges.ErrNotSupported
}

func (s *repoService) ListTags(ctx context.Context, owner, repo string) ([]forges.Tag, error) {
	repoDID, err := s.f.repoDID(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	var tags []forges.Tag
	cursor := ""
	for {
		params := url.Values{}
		params.Set("repo", repoDID)
		addLimit(params, 100)
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		var raw any
		if err := s.f.xrpc(ctx, xrpcListTags, params, &raw); err != nil {
			return nil, err
		}
		items, next := collection(raw, "tags", "refs", "values")
		for _, item := range items {
			switch v := item.(type) {
			case string:
				tags = append(tags, forges.Tag{Name: v})
			case map[string]any:
				name := stringField(v, "name", "ref")
				if name == "" {
					continue
				}
				tags = append(tags, forges.Tag{
					Name:   strings.TrimPrefix(name, "refs/tags/"),
					Commit: stringField(v, "sha", "oid", "target", "commit", "hash"),
				})
			}
		}
		if next == "" {
			return tags, nil
		}
		cursor = next
	}
}

func (s *repoService) ListContributors(context.Context, string, string) ([]forges.Contributor, error) {
	return nil, forges.ErrNotSupported
}

func (s *repoService) Search(context.Context, forges.SearchRepoOpts) ([]forges.Repository, error) {
	return nil, forges.ErrNotSupported
}

func (s *repoService) SettingsURL(repoHTMLURL string) string {
	return strings.TrimRight(repoHTMLURL, "/") + "/settings"
}
func (s *repoService) WikiURL(repoHTMLURL string) string {
	return strings.TrimRight(repoHTMLURL, "/") + "/wiki"
}
func (s *repoService) ActionsURL(repoHTMLURL string) string {
	return strings.TrimRight(repoHTMLURL, "/") + "/pipelines"
}
func (s *repoService) ReleasesURL(repoHTMLURL string) string {
	return strings.TrimRight(repoHTMLURL, "/") + "/releases"
}
func (s *repoService) BlobURL(repoHTMLURL, ref, filePath string) string {
	return strings.TrimRight(repoHTMLURL, "/") + "/blob/" + url.PathEscape(ref) + "/" + strings.TrimLeft(filePath, "/")
}

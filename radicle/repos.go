package radicle

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	forges "github.com/git-pkgs/forge"
)

// repoInfo is the subset of GET /api/v1/repos/:rid that the backend uses.
type repoInfo struct {
	RID      string `json:"rid"`
	Payloads struct {
		Project *struct {
			Data struct {
				Name          string `json:"name"`
				Description   string `json:"description"`
				DefaultBranch string `json:"defaultBranch"`
			} `json:"data"`
			Meta struct {
				Head   string `json:"head"`
				Issues struct {
					Open int `json:"open"`
				} `json:"issues"`
			} `json:"meta"`
		} `json:"xyz.radicle.project"`
	} `json:"payloads"`
	Visibility struct {
		Type string `json:"type"`
	} `json:"visibility"`
	Refs *struct {
		// Refs maps canonical ref names such as refs/heads/main to commit SHAs.
		Refs map[string]string `json:"refs"`
		// Tags maps tag ref names to either a commit SHA or an annotated tag
		// object with a commit field.
		Tags map[string]json.RawMessage `json:"tags"`
	} `json:"refs"`
}

func (r *repoInfo) name() string {
	if r.Payloads.Project != nil {
		return r.Payloads.Project.Data.Name
	}
	return ""
}

func (r *repoInfo) description() string {
	if r.Payloads.Project != nil {
		return r.Payloads.Project.Data.Description
	}
	return ""
}

func (r *repoInfo) defaultBranch() string {
	if r.Payloads.Project != nil {
		return r.Payloads.Project.Data.DefaultBranch
	}
	return ""
}

func (r *repoInfo) head() string {
	if r.Payloads.Project != nil {
		return r.Payloads.Project.Meta.Head
	}
	return ""
}

// branches returns the canonical branches sorted by name. Nodes that predate
// the canonical refs field only report the default branch head.
func (r *repoInfo) branches() []forges.Branch {
	def := r.defaultBranch()
	var out []forges.Branch
	if r.Refs != nil {
		for ref, sha := range r.Refs.Refs {
			name, ok := strings.CutPrefix(ref, "refs/heads/")
			if !ok || name == "" {
				continue
			}
			out = append(out, forges.Branch{Name: name, SHA: sha, Default: name == def})
		}
	}
	if len(out) == 0 && def != "" && r.head() != "" {
		out = append(out, forges.Branch{Name: def, SHA: r.head(), Default: true})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (r *repoInfo) tags() []forges.Tag {
	if r.Refs == nil {
		return nil
	}
	out := make([]forges.Tag, 0, len(r.Refs.Tags))
	for ref, raw := range r.Refs.Tags {
		name := strings.TrimPrefix(ref, "refs/tags/")
		if name == "" {
			continue
		}
		out = append(out, forges.Tag{Name: name, Commit: tagCommit(raw)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func tagCommit(raw json.RawMessage) string {
	var sha string
	if json.Unmarshal(raw, &sha) == nil {
		return sha
	}
	var obj struct {
		Commit string `json:"commit"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		return obj.Commit
	}
	return ""
}

// resolveRef turns a branch name, tag name or commit SHA into the commit SHA
// that radicle-httpd's tree and blob endpoints expect. An empty ref or HEAD
// means the head of the default branch.
func (r *repoInfo) resolveRef(ref string) (string, error) {
	if ref == "" || ref == "HEAD" {
		if h := r.head(); h != "" {
			return h, nil
		}
		return "", fmt.Errorf("radicle repository %s has no default branch head", r.RID)
	}
	if forges.IsFullCommitSHA(ref) {
		return strings.ToLower(ref), nil
	}
	if r.Refs != nil {
		for _, name := range []string{ref, "refs/heads/" + ref} {
			if sha, ok := r.Refs.Refs[name]; ok && sha != "" {
				return sha, nil
			}
		}
		for _, name := range []string{ref, "refs/tags/" + ref} {
			if raw, ok := r.Refs.Tags[name]; ok {
				if sha := tagCommit(raw); sha != "" {
					return sha, nil
				}
			}
		}
	}
	if ref == r.defaultBranch() && r.head() != "" {
		return r.head(), nil
	}
	return "", fmt.Errorf("ref %q in %s: %w", ref, r.RID, forges.ErrNotFound)
}

func (f *radicleForge) repoInfo(ctx context.Context, owner, repo string) (*repoInfo, error) {
	id, err := rid(owner, repo)
	if err != nil {
		return nil, err
	}
	var info repoInfo
	if err := f.getJSON(ctx, f.repoPath(id), &info); err != nil {
		return nil, err
	}
	if info.RID == "" {
		info.RID = id
	}
	return &info, nil
}

// cloneURL returns the git-over-HTTP URL radicle-httpd serves a repository
// at, which is the RID without its rad: prefix plus .git.
func (f *radicleForge) cloneURL(id string) string {
	return f.baseURL + "/" + strings.TrimPrefix(id, RIDPrefix) + ".git"
}

type repoService struct {
	f *radicleForge
}

func (s *repoService) Get(ctx context.Context, owner, repo string) (*forges.Repository, error) {
	info, err := s.f.repoInfo(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	name := info.name()
	if name == "" {
		name = info.RID
	}
	result := &forges.Repository{
		FullName:            info.RID,
		Name:                name,
		Description:         info.description(),
		CloneURL:            s.f.cloneURL(info.RID),
		DefaultBranch:       info.defaultBranch(),
		Private:             info.Visibility.Type == "private",
		HasIssues:           true,
		PullRequestsEnabled: true,
	}
	if info.Payloads.Project != nil {
		result.OpenIssuesCount = info.Payloads.Project.Meta.Issues.Open
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
	info, err := s.f.repoInfo(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	return info.tags(), nil
}

func (s *repoService) ListContributors(context.Context, string, string) ([]forges.Contributor, error) {
	return nil, forges.ErrNotSupported
}

func (s *repoService) Search(context.Context, forges.SearchRepoOpts) ([]forges.Repository, error) {
	return nil, forges.ErrNotSupported
}

// radicle-httpd has no web interface, so there are no pages to link to.
// These return the repository URL unchanged.

func (s *repoService) SettingsURL(repoHTMLURL string) string   { return repoHTMLURL }
func (s *repoService) WikiURL(repoHTMLURL string) string       { return repoHTMLURL }
func (s *repoService) ActionsURL(repoHTMLURL string) string    { return repoHTMLURL }
func (s *repoService) ReleasesURL(repoHTMLURL string) string   { return repoHTMLURL }
func (s *repoService) BlobURL(repoHTMLURL, _, _ string) string { return repoHTMLURL }

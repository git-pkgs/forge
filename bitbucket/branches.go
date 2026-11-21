package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	forge "github.com/git-pkgs/forge"
)

type bitbucketBranchService struct {
	token      string
	httpClient *http.Client
}

func (f *bitbucketForge) Branches() forge.BranchService {
	return &bitbucketBranchService{token: f.token, httpClient: f.httpClient}
}

type bbBranch struct {
	Name   string `json:"name"`
	Target struct {
		Hash string `json:"hash"`
	} `json:"target"`
}

// convertBitbucketBranch maps a Bitbucket branch. Branch objects don't say
// whether the branch is the main branch or has restrictions, so Default and
// Protected stay false.
func convertBitbucketBranch(bb bbBranch) forge.Branch {
	return forge.Branch{
		Name: bb.Name,
		SHA:  bb.Target.Hash,
	}
}

func (s *bitbucketBranchService) doJSON(ctx context.Context, method, url string, body any, v any) error {
	rs := &bitbucketRepoService{token: s.token, httpClient: s.httpClient}
	return rs.doJSON(ctx, method, url, body, v)
}

func (s *bitbucketBranchService) List(ctx context.Context, owner, repo string, opts forge.ListBranchOpts) ([]forge.Branch, error) {
	u := fmt.Sprintf("%s/repositories/%s/%s/refs/branches?%s", bitbucketAPI, owner, repo, pageQuery(opts.Page, opts.PerPage, opts.Limit))
	rs := &bitbucketRepoService{token: s.token, httpClient: s.httpClient}
	return listPages(ctx, rs, u, opts.Limit, convertBitbucketBranch)
}

// Create creates branch name at the head of branch from, or at from itself
// when it is a full commit SHA. Bitbucket's create endpoint only takes a
// commit hash as the target, so a branch is looked up first.
func (s *bitbucketBranchService) Create(ctx context.Context, owner, repo, name, from string) (*forge.Branch, error) {
	target := from
	if !forge.IsFullCommitSHA(from) {
		var src bbBranch
		if err := s.doJSON(ctx, http.MethodGet, branchURL(owner, repo, from), nil, &src); err != nil {
			return nil, err
		}
		target = src.Target.Hash
	}

	body := map[string]any{
		"name":   name,
		"target": map[string]string{"hash": target},
	}
	u := fmt.Sprintf("%s/repositories/%s/%s/refs/branches", bitbucketAPI, owner, repo)
	var bb bbBranch
	if err := s.doJSON(ctx, http.MethodPost, u, body, &bb); err != nil {
		return nil, err
	}
	result := convertBitbucketBranch(bb)
	return &result, nil
}

func (s *bitbucketBranchService) Delete(ctx context.Context, owner, repo, name string) error {
	return s.doJSON(ctx, http.MethodDelete, branchURL(owner, repo, name), nil, nil)
}

// branchURL returns the URL of a single branch. Branch names can contain
// slashes, which Bitbucket only accepts escaped into one path segment.
func branchURL(owner, repo, name string) string {
	return fmt.Sprintf("%s/repositories/%s/%s/refs/branches/%s", bitbucketAPI, owner, repo, url.PathEscape(name))
}

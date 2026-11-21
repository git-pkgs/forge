package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	forge "github.com/git-pkgs/forge"
)

// maxBranchPageLen is the largest page Bitbucket serves from the refs
// endpoints.
const maxBranchPageLen = 100

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

type bbBranchesResponse struct {
	Values []bbBranch `json:"values"`
	Next   string     `json:"next"`
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
	perPage := opts.PerPage
	if perPage <= 0 || perPage > maxBranchPageLen {
		perPage = maxBranchPageLen
	}
	// A start page is counted in pages of perPage, so the page size can only
	// shrink to the limit when listing from the first page.
	if opts.Limit > 0 && perPage > opts.Limit && opts.Page <= 1 {
		perPage = opts.Limit
	}

	u := fmt.Sprintf("%s/repositories/%s/%s/refs/branches?pagelen=%d", bitbucketAPI, owner, repo, perPage)
	if opts.Page > 1 {
		u += fmt.Sprintf("&page=%d", opts.Page)
	}

	var all []forge.Branch
	for u != "" {
		var page bbBranchesResponse
		if err := s.doJSON(ctx, http.MethodGet, u, nil, &page); err != nil {
			return nil, err
		}
		for _, bb := range page.Values {
			all = append(all, convertBitbucketBranch(bb))
		}
		if opts.Limit > 0 && len(all) >= opts.Limit {
			break
		}
		u = page.Next
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
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

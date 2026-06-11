package tangled

import (
	"context"
	"net/url"
	"strings"

	forges "github.com/git-pkgs/forge"
)

type branchService struct {
	f *tangledForge
}

func (s *branchService) List(ctx context.Context, owner, repo string, opts forges.ListBranchOpts) ([]forges.Branch, error) {
	repoDID, err := s.f.repoDID(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	var branches []forges.Branch
	cursor := ""
	for {
		params := url.Values{}
		params.Set("repo", repoDID)
		addLimit(params, perPage(opts.Limit, opts.PerPage))
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		var raw any
		if err := s.f.xrpc(ctx, xrpcListBranches, params, &raw); err != nil {
			return nil, err
		}
		items, next := collection(raw, "branches", "refs", "values")
		for _, item := range items {
			switch v := item.(type) {
			case string:
				branches = append(branches, forges.Branch{Name: strings.TrimPrefix(v, "refs/heads/")})
			case map[string]any:
				name := stringField(v, "name", "ref")
				if name == "" {
					continue
				}
				branches = append(branches, forges.Branch{
					Name:      strings.TrimPrefix(name, "refs/heads/"),
					SHA:       stringField(v, "sha", "oid", "target", "commit", "hash"),
					Default:   boolField(v, "default", "isDefault"),
					Protected: boolField(v, "protected"),
				})
			}
			if limitReached(len(branches), opts.Limit) {
				return branches, nil
			}
		}
		if next == "" {
			return branches, nil
		}
		cursor = next
	}
}

func (s *branchService) Create(context.Context, string, string, string, string) (*forges.Branch, error) {
	return nil, forges.ErrNotSupported
}

func (s *branchService) Delete(context.Context, string, string, string) error {
	return forges.ErrNotSupported
}

package radicle

import (
	"context"

	forges "github.com/git-pkgs/forge"
)

type branchService struct {
	f *radicleForge
}

// List returns the repository's canonical branches, which radicle-httpd
// reports on the repository itself, so a single request covers every page.
func (s *branchService) List(ctx context.Context, owner, repo string, opts forges.ListBranchOpts) ([]forges.Branch, error) {
	info, err := s.f.repoInfo(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	branches := info.branches()
	if opts.Limit > 0 && len(branches) > opts.Limit {
		branches = branches[:opts.Limit]
	}
	return branches, nil
}

func (s *branchService) Create(context.Context, string, string, string, string) (*forges.Branch, error) {
	return nil, forges.ErrNotSupported
}

func (s *branchService) Delete(context.Context, string, string, string) error {
	return forges.ErrNotSupported
}

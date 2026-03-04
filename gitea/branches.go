package gitea

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

type giteaBranchService struct {
	client *gitea.Client
}

func (f *giteaForge) Branches() forge.BranchService {
	return &giteaBranchService{client: f.client}
}

func (s *giteaBranchService) List(ctx context.Context, owner, repo string, opts forge.ListBranchOpts) ([]forge.Branch, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	var all []forge.Branch
	for {
		branches, resp, err := s.client.ListRepoBranches(owner, repo, gitea.ListRepoBranchesOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, b := range branches {
			branch := forge.Branch{
				Name:      b.Name,
				Protected: b.Protected,
			}
			if b.Commit != nil {
				branch.SHA = b.Commit.ID
			}
			all = append(all, branch)
		}
		if len(branches) < perPage || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		page++
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *giteaBranchService) Create(ctx context.Context, owner, repo, name, from string) (*forge.Branch, error) {
	b, resp, err := s.client.CreateBranch(owner, repo, gitea.CreateBranchOption{
		BranchName:    name,
		OldBranchName: from,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	result := forge.Branch{
		Name: b.Name,
	}
	if b.Commit != nil {
		result.SHA = b.Commit.ID
	}
	return &result, nil
}

func (s *giteaBranchService) Delete(ctx context.Context, owner, repo, name string) error {
	_, resp, err := s.client.DeleteRepoBranch(owner, repo, name)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

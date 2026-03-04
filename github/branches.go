package github

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"github.com/google/go-github/v82/github"
)

type gitHubBranchService struct {
	client *github.Client
}

func (f *gitHubForge) Branches() forge.BranchService {
	return &gitHubBranchService{client: f.client}
}

func (s *gitHubBranchService) List(ctx context.Context, owner, repo string, opts forge.ListBranchOpts) ([]forge.Branch, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.BranchListOptions{
		ListOptions: github.ListOptions{PerPage: perPage, Page: page},
	}

	var all []forge.Branch
	for {
		branches, resp, err := s.client.Repositories.ListBranches(ctx, owner, repo, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, b := range branches {
			branch := forge.Branch{
				Name:      b.GetName(),
				Protected: b.GetProtected(),
			}
			if c := b.GetCommit(); c != nil {
				branch.SHA = c.GetSHA()
			}
			all = append(all, branch)
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

func (s *gitHubBranchService) Create(ctx context.Context, owner, repo, name, from string) (*forge.Branch, error) {
	// Get the SHA of the source ref
	ref, resp, err := s.client.Git.GetRef(ctx, owner, repo, "refs/heads/"+from)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	created, resp, err := s.client.Git.CreateRef(ctx, owner, repo, github.CreateRef{
		Ref: "refs/heads/" + name,
		SHA: ref.GetObject().GetSHA(),
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	result := forge.Branch{
		Name: name,
		SHA:  created.GetObject().GetSHA(),
	}
	return &result, nil
}

func (s *gitHubBranchService) Delete(ctx context.Context, owner, repo, name string) error {
	resp, err := s.client.Git.DeleteRef(ctx, owner, repo, "refs/heads/"+name)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

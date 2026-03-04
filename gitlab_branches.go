package forges

import (
	"context"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabBranchService struct {
	client *gitlab.Client
}

func (f *gitLabForge) Branches() BranchService {
	return &gitLabBranchService{client: f.client}
}

func (s *gitLabBranchService) List(ctx context.Context, owner, repo string, opts ListBranchOpts) ([]Branch, error) {
	pid := owner + "/" + repo
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	glOpts := &gitlab.ListBranchesOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage), Page: int64(page)},
	}

	var all []Branch
	for {
		branches, resp, err := s.client.Branches.ListBranches(pid, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		for _, b := range branches {
			branch := Branch{
				Name:      b.Name,
				Protected: b.Protected,
				Default:   b.Default,
			}
			if b.Commit != nil {
				branch.SHA = b.Commit.ID
			}
			all = append(all, branch)
		}
		if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		glOpts.Page = int64(resp.NextPage)
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *gitLabBranchService) Create(ctx context.Context, owner, repo, name, from string) (*Branch, error) {
	pid := owner + "/" + repo
	glOpts := &gitlab.CreateBranchOptions{
		Branch: gitlab.Ptr(name),
		Ref:    gitlab.Ptr(from),
	}

	b, resp, err := s.client.Branches.CreateBranch(pid, glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	result := Branch{
		Name:      b.Name,
		Protected: b.Protected,
	}
	if b.Commit != nil {
		result.SHA = b.Commit.ID
	}
	return &result, nil
}

func (s *gitLabBranchService) Delete(ctx context.Context, owner, repo, name string) error {
	pid := owner + "/" + repo
	resp, err := s.client.Branches.DeleteBranch(pid, name)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

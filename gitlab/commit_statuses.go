package gitlab

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const defaultPageSize = 100

type gitLabCommitStatusService struct {
	client *gitlab.Client
}

func (f *gitLabForge) CommitStatuses() forge.CommitStatusService {
	return &gitLabCommitStatusService{client: f.client}
}

func (s *gitLabCommitStatusService) List(ctx context.Context, owner, repo, sha string) ([]forge.CommitStatus, error) {
	pid := owner + "/" + repo
	var all []forge.CommitStatus
	opts := &gitlab.GetCommitStatusesOptions{
		ListOptions: gitlab.ListOptions{PerPage: defaultPageSize},
	}
	for {
		statuses, resp, err := s.client.Commits.GetCommitStatuses(pid, sha, opts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, st := range statuses {
			cs := forge.CommitStatus{
				State:       st.Status,
				Context:     st.Name,
				Description: st.Description,
				TargetURL:   st.TargetURL,
				Creator:     st.Author.Username,
			}
			if st.CreatedAt != nil {
				cs.CreatedAt = *st.CreatedAt
			}
			all = append(all, cs)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

func (s *gitLabCommitStatusService) Set(ctx context.Context, owner, repo, sha string, opts forge.SetCommitStatusOpts) (*forge.CommitStatus, error) {
	pid := owner + "/" + repo
	glOpts := &gitlab.SetCommitStatusOptions{
		State:       gitlab.BuildStateValue(opts.State),
		Name:        gitlab.Ptr(opts.Context),
		Description: gitlab.Ptr(opts.Description),
		TargetURL:   gitlab.Ptr(opts.TargetURL),
	}

	result, resp, err := s.client.Commits.SetCommitStatus(pid, sha, glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	cs := &forge.CommitStatus{
		State:       result.Status,
		Context:     result.Name,
		Description: result.Description,
		TargetURL:   result.TargetURL,
		Creator:     result.Author.Username,
	}
	if result.CreatedAt != nil {
		cs.CreatedAt = *result.CreatedAt
	}
	return cs, nil
}

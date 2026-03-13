package github

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"github.com/google/go-github/v82/github"
)

type gitHubCommitStatusService struct {
	client *github.Client
}

func (f *gitHubForge) CommitStatuses() forge.CommitStatusService {
	return &gitHubCommitStatusService{client: f.client}
}

func (s *gitHubCommitStatusService) List(ctx context.Context, owner, repo, sha string) ([]forge.CommitStatus, error) {
	var all []forge.CommitStatus
	opts := &github.ListOptions{PerPage: 100}
	for {
		statuses, resp, err := s.client.Repositories.ListStatuses(ctx, owner, repo, sha, opts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, st := range statuses {
			cs := forge.CommitStatus{
				State:       st.GetState(),
				Context:     st.GetContext(),
				Description: st.GetDescription(),
				TargetURL:   st.GetTargetURL(),
			}
			if st.Creator != nil {
				cs.Creator = st.Creator.GetLogin()
			}
			if st.CreatedAt != nil {
				cs.CreatedAt = st.CreatedAt.Time
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

func (s *gitHubCommitStatusService) Set(ctx context.Context, owner, repo, sha string, opts forge.SetCommitStatusOpts) (*forge.CommitStatus, error) {
	status := github.RepoStatus{
		State:       &opts.State,
		Context:     &opts.Context,
		Description: &opts.Description,
		TargetURL:   &opts.TargetURL,
	}

	result, resp, err := s.client.Repositories.CreateStatus(ctx, owner, repo, sha, status)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	cs := &forge.CommitStatus{
		State:       result.GetState(),
		Context:     result.GetContext(),
		Description: result.GetDescription(),
		TargetURL:   result.GetTargetURL(),
	}
	if result.Creator != nil {
		cs.Creator = result.Creator.GetLogin()
	}
	if result.CreatedAt != nil {
		cs.CreatedAt = result.CreatedAt.Time
	}
	return cs, nil
}

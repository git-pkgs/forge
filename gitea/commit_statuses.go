package gitea

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

const defaultPageSize = 50

type giteaCommitStatusService struct {
	client *gitea.Client
}

func (f *giteaForge) CommitStatuses() forge.CommitStatusService {
	return &giteaCommitStatusService{client: f.client}
}

func (s *giteaCommitStatusService) List(ctx context.Context, owner, repo, sha string) ([]forge.CommitStatus, error) {
	var all []forge.CommitStatus
	page := 1
	for {
		statuses, resp, err := s.client.ListStatuses(owner, repo, sha, gitea.ListStatusesOption{
			ListOptions: gitea.ListOptions{Page: page, PageSize: defaultPageSize},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, st := range statuses {
			cs := forge.CommitStatus{
				State:       string(st.State),
				Context:     st.Context,
				Description: st.Description,
				TargetURL:   st.TargetURL,
				CreatedAt:   st.Created,
			}
			if st.Creator != nil {
				cs.Creator = st.Creator.UserName
			}
			all = append(all, cs)
		}
		if len(statuses) < defaultPageSize {
			break
		}
		page++
	}
	return all, nil
}

func (s *giteaCommitStatusService) Set(ctx context.Context, owner, repo, sha string, opts forge.SetCommitStatusOpts) (*forge.CommitStatus, error) {
	result, resp, err := s.client.CreateStatus(owner, repo, sha, gitea.CreateStatusOption{
		State:       gitea.StatusState(opts.State),
		TargetURL:   opts.TargetURL,
		Description: opts.Description,
		Context:     opts.Context,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	cs := &forge.CommitStatus{
		State:       string(result.State),
		Context:     result.Context,
		Description: result.Description,
		TargetURL:   result.TargetURL,
		CreatedAt:   result.Created,
	}
	if result.Creator != nil {
		cs.Creator = result.Creator.UserName
	}
	return cs, nil
}

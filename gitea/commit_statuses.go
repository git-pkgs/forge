package gitea

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

const defaultPageSize = 50

// pageSize caps the requested page size at Gitea's default MAX_RESPONSE_ITEMS.
// Servers clamp larger values, which breaks len(results) < perPage loop exits.
func pageSize(perPage int) int {
	if perPage <= 0 || perPage > defaultPageSize {
		return defaultPageSize
	}
	return perPage
}

// lastPage reports whether a paginated response was the final page. It trusts
// the SDK-parsed Link headers when the server sent any, since Gitea clamps the
// page size to MAX_RESPONSE_ITEMS and a clamped page would otherwise look like
// a short final page. Falls back to the short-page heuristic when no Link
// header was sent.
func lastPage(resp *gitea.Response, got, perPage int) bool {
	if got == 0 {
		return true
	}
	if resp != nil && (resp.FirstPage > 0 || resp.PrevPage > 0 || resp.NextPage > 0 || resp.LastPage > 0) {
		return resp.NextPage == 0
	}
	return got < perPage
}

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

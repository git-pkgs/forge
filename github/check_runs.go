package github

import (
	"context"
	"net/http"

	forge "github.com/git-pkgs/forge"

	"github.com/google/go-github/v82/github"
)

type gitHubCheckRunService struct {
	client *github.Client
}

func (f *gitHubForge) CheckRuns() forge.CheckRunService {
	return &gitHubCheckRunService{client: f.client}
}

func (s *gitHubCheckRunService) List(ctx context.Context, owner, repo, sha string) ([]forge.CheckRun, error) {
	var all []forge.CheckRun
	opts := &github.ListCheckRunsOptions{
		ListOptions: github.ListOptions{PerPage: defaultPageSize},
	}
	for {
		result, resp, err := s.client.Checks.ListCheckRunsForRef(ctx, owner, repo, sha, opts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, wrapErr(err)
		}
		for _, r := range result.CheckRuns {
			cr := forge.CheckRun{
				ID:         r.GetID(),
				Name:       r.GetName(),
				Status:     forge.NormalizeCIStatus(r.GetStatus()),
				Conclusion: forge.NormalizeCIConclusion(r.GetConclusion()),
				HTMLURL:    r.GetHTMLURL(),
			}
			if r.StartedAt != nil {
				t := r.StartedAt.Time
				cr.StartedAt = &t
			}
			if r.CompletedAt != nil {
				t := r.CompletedAt.Time
				cr.CompletedAt = &t
			}
			all = append(all, cr)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

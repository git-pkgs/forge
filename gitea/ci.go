package gitea

import (
	"bytes"
	"context"
	"io"
	"net/http"

	forge "github.com/git-pkgs/forge"

	"code.gitea.io/sdk/gitea"
)

type giteaCIService struct {
	client *gitea.Client
}

func (f *giteaForge) CI() forge.CIService {
	return &giteaCIService{client: f.client}
}

func convertGiteaWorkflowRun(r *gitea.ActionWorkflowRun) forge.CIRun {
	result := forge.CIRun{
		ID:        r.ID,
		Title:     r.DisplayTitle,
		Status:    r.Status,
		Branch:    r.HeadBranch,
		SHA:       r.HeadSha,
		Event:     r.Event,
		HTMLURL:   r.HTMLURL,
		CreatedAt: r.StartedAt,
	}

	if r.Conclusion != "" {
		result.Conclusion = r.Conclusion
	}

	if r.Actor != nil {
		result.Author = forge.User{
			Login:     r.Actor.UserName,
			AvatarURL: r.Actor.AvatarURL,
		}
	}

	if !r.CompletedAt.IsZero() {
		t := r.CompletedAt
		result.FinishedAt = &t
	}

	return result
}

func convertGiteaWorkflowJob(j *gitea.ActionWorkflowJob) forge.CIJob {
	job := forge.CIJob{
		ID:         j.ID,
		Name:       j.Name,
		Status:     j.Status,
		Conclusion: j.Conclusion,
		HTMLURL:    j.HTMLURL,
	}
	if !j.StartedAt.IsZero() {
		t := j.StartedAt
		job.StartedAt = &t
	}
	if !j.CompletedAt.IsZero() {
		t := j.CompletedAt
		job.FinishedAt = &t
	}
	return job
}

func (s *giteaCIService) ListRuns(_ context.Context, owner, repo string, opts forge.ListCIRunOpts) ([]forge.CIRun, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	gOpts := gitea.ListRepoActionRunsOptions{
		ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
	}
	if opts.Branch != "" {
		gOpts.Branch = opts.Branch
	}
	if opts.Status != "" {
		gOpts.Status = opts.Status
	}
	if opts.User != "" {
		gOpts.Actor = opts.User
	}

	var all []forge.CIRun
	for {
		resp, httpResp, err := s.client.ListRepoActionRuns(owner, repo, gOpts)
		if err != nil {
			if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, r := range resp.WorkflowRuns {
			all = append(all, convertGiteaWorkflowRun(r))
		}
		if len(resp.WorkflowRuns) < perPage || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		gOpts.Page++
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *giteaCIService) GetRun(_ context.Context, owner, repo string, runID int64) (*forge.CIRun, error) {
	r, resp, err := s.client.GetRepoActionRun(owner, repo, runID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaWorkflowRun(r)

	jobs, _, err := s.client.ListRepoActionRunJobs(owner, repo, runID, gitea.ListRepoActionJobsOptions{})
	if err == nil {
		for _, j := range jobs.Jobs {
			result.Jobs = append(result.Jobs, convertGiteaWorkflowJob(j))
		}
	}

	return &result, nil
}

func (s *giteaCIService) TriggerRun(_ context.Context, _, _ string, _ forge.TriggerCIRunOpts) error {
	return forge.ErrNotSupported
}

func (s *giteaCIService) CancelRun(_ context.Context, _, _ string, _ int64) error {
	return forge.ErrNotSupported
}

func (s *giteaCIService) RetryRun(_ context.Context, _, _ string, _ int64) error {
	return forge.ErrNotSupported
}

func (s *giteaCIService) GetJobLog(_ context.Context, owner, repo string, jobID int64) (io.ReadCloser, error) {
	data, resp, err := s.client.GetRepoActionJobLogs(owner, repo, jobID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

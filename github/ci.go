package github

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"io"
	"net/http"

	"github.com/google/go-github/v82/github"
)

const maxLogRedirects = 4

type gitHubCIService struct {
	client *github.Client
}

func (f *gitHubForge) CI() forge.CIService {
	return &gitHubCIService{client: f.client}
}

func convertGitHubWorkflowRun(r *github.WorkflowRun) forge.CIRun {
	result := forge.CIRun{
		ID:      r.GetID(),
		Title:   r.GetName(),
		Status:  r.GetStatus(),
		Branch:  r.GetHeadBranch(),
		SHA:     r.GetHeadSHA(),
		Event:   r.GetEvent(),
		HTMLURL: r.GetHTMLURL(),
	}

	if c := r.GetConclusion(); c != "" {
		result.Conclusion = c
	}

	if a := r.GetActor(); a != nil {
		result.Author = forge.User{
			Login:     a.GetLogin(),
			AvatarURL: a.GetAvatarURL(),
		}
	}

	if t := r.GetCreatedAt(); !t.IsZero() {
		result.CreatedAt = t.Time
	}
	if t := r.GetUpdatedAt(); !t.IsZero() {
		result.UpdatedAt = t.Time
	}

	return result
}

func (s *gitHubCIService) ListRuns(ctx context.Context, owner, repo string, opts forge.ListCIRunOpts) ([]forge.CIRun, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{PerPage: perPage, Page: page},
	}
	if opts.Branch != "" {
		ghOpts.Branch = opts.Branch
	}
	if opts.Status != "" {
		ghOpts.Status = opts.Status
	}
	if opts.User != "" {
		ghOpts.Actor = opts.User
	}

	var all []forge.CIRun
	for {
		var runs *github.WorkflowRuns
		var resp *github.Response
		var err error

		if opts.Workflow != "" {
			runs, resp, err = s.client.Actions.ListWorkflowRunsByFileName(ctx, owner, repo, opts.Workflow, ghOpts)
		} else {
			runs, resp, err = s.client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, ghOpts)
		}
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, r := range runs.WorkflowRuns {
			all = append(all, convertGitHubWorkflowRun(r))
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

func (s *gitHubCIService) GetRun(ctx context.Context, owner, repo string, runID int64) (*forge.CIRun, error) {
	r, resp, err := s.client.Actions.GetWorkflowRunByID(ctx, owner, repo, runID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubWorkflowRun(r)

	// Fetch jobs for this run
	jobs, _, err := s.client.Actions.ListWorkflowJobs(ctx, owner, repo, runID, &github.ListWorkflowJobsOptions{})
	if err == nil {
		for _, j := range jobs.Jobs {
			job := forge.CIJob{
				ID:         j.GetID(),
				Name:       j.GetName(),
				Status:     j.GetStatus(),
				Conclusion: j.GetConclusion(),
				HTMLURL:    j.GetHTMLURL(),
			}
			if t := j.GetStartedAt(); !t.IsZero() {
				st := t.Time
				job.StartedAt = &st
			}
			if t := j.GetCompletedAt(); !t.IsZero() {
				ft := t.Time
				job.FinishedAt = &ft
			}
			result.Jobs = append(result.Jobs, job)
		}
	}

	return &result, nil
}

func (s *gitHubCIService) TriggerRun(ctx context.Context, owner, repo string, opts forge.TriggerCIRunOpts) error {
	event := github.CreateWorkflowDispatchEventRequest{
		Ref: opts.Branch,
	}
	if len(opts.Inputs) > 0 {
		inputs := make(map[string]any, len(opts.Inputs))
		for k, v := range opts.Inputs {
			inputs[k] = v
		}
		event.Inputs = inputs
	}

	resp, err := s.client.Actions.CreateWorkflowDispatchEventByFileName(ctx, owner, repo, opts.Workflow, event)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitHubCIService) CancelRun(ctx context.Context, owner, repo string, runID int64) error {
	resp, err := s.client.Actions.CancelWorkflowRunByID(ctx, owner, repo, runID)
	if err != nil {
		if _, ok := err.(*github.AcceptedError); ok {
			return nil
		}
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitHubCIService) RetryRun(ctx context.Context, owner, repo string, runID int64) error {
	resp, err := s.client.Actions.RerunWorkflowByID(ctx, owner, repo, runID)
	if err != nil {
		if _, ok := err.(*github.AcceptedError); ok {
			return nil
		}
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitHubCIService) GetJobLog(ctx context.Context, owner, repo string, jobID int64) (io.ReadCloser, error) {
	url, resp, err := s.client.Actions.GetWorkflowJobLogs(ctx, owner, repo, jobID, maxLogRedirects)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	// url is a redirect URL; fetch the actual log
	logResp, err := http.Get(url.String())
	if err != nil {
		return nil, err
	}
	return logResp.Body, nil
}

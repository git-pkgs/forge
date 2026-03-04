package gitlab

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"io"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabCIService struct {
	client *gitlab.Client
}

func (f *gitLabForge) CI() forge.CIService {
	return &gitLabCIService{client: f.client}
}

func convertGitLabPipeline(p *gitlab.PipelineInfo) forge.CIRun {
	result := forge.CIRun{
		ID:      int64(p.ID),
		Status:  p.Status,
		Branch:  p.Ref,
		SHA:     p.SHA,
		HTMLURL: p.WebURL,
	}
	if p.CreatedAt != nil {
		result.CreatedAt = *p.CreatedAt
	}
	if p.UpdatedAt != nil {
		result.UpdatedAt = *p.UpdatedAt
	}
	return result
}

func convertGitLabPipelineDetail(p *gitlab.Pipeline) forge.CIRun {
	result := forge.CIRun{
		ID:      int64(p.ID),
		Status:  p.Status,
		Branch:  p.Ref,
		SHA:     p.SHA,
		HTMLURL: p.WebURL,
	}
	if p.User != nil {
		result.Author = forge.User{
			Login:     p.User.Username,
			AvatarURL: p.User.AvatarURL,
		}
	}
	if p.CreatedAt != nil {
		result.CreatedAt = *p.CreatedAt
	}
	if p.UpdatedAt != nil {
		result.UpdatedAt = *p.UpdatedAt
	}
	if p.FinishedAt != nil {
		result.FinishedAt = p.FinishedAt
	}
	return result
}

func (s *gitLabCIService) ListRuns(ctx context.Context, owner, repo string, opts forge.ListCIRunOpts) ([]forge.CIRun, error) {
	pid := owner + "/" + repo
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 20
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	glOpts := &gitlab.ListProjectPipelinesOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage), Page: int64(page)},
	}
	if opts.Branch != "" {
		glOpts.Ref = gitlab.Ptr(opts.Branch)
	}
	if opts.Status != "" {
		glOpts.Status = gitlab.Ptr(gitlab.BuildStateValue(opts.Status))
	}
	if opts.User != "" {
		glOpts.Username = gitlab.Ptr(opts.User)
	}

	var all []forge.CIRun
	for {
		pipelines, resp, err := s.client.Pipelines.ListProjectPipelines(pid, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, p := range pipelines {
			all = append(all, convertGitLabPipeline(p))
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

func (s *gitLabCIService) GetRun(ctx context.Context, owner, repo string, runID int64) (*forge.CIRun, error) {
	pid := owner + "/" + repo
	p, resp, err := s.client.Pipelines.GetPipeline(pid, runID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabPipelineDetail(p)

	// Fetch jobs
	jobs, _, err := s.client.Jobs.ListPipelineJobs(pid, runID, &gitlab.ListJobsOptions{})
	if err == nil {
		for _, j := range jobs {
			job := forge.CIJob{
				ID:      int64(j.ID),
				Name:    j.Name,
				Status:  j.Status,
				HTMLURL: j.WebURL,
			}
			if j.StartedAt != nil {
				job.StartedAt = j.StartedAt
			}
			if j.FinishedAt != nil {
				job.FinishedAt = j.FinishedAt
			}
			result.Jobs = append(result.Jobs, job)
		}
	}

	return &result, nil
}

func (s *gitLabCIService) TriggerRun(ctx context.Context, owner, repo string, opts forge.TriggerCIRunOpts) error {
	pid := owner + "/" + repo
	glOpts := &gitlab.CreatePipelineOptions{
		Ref: gitlab.Ptr(opts.Branch),
	}
	if len(opts.Inputs) > 0 {
		vars := make([]*gitlab.PipelineVariableOptions, 0, len(opts.Inputs))
		for k, v := range opts.Inputs {
			vars = append(vars, &gitlab.PipelineVariableOptions{
				Key:   gitlab.Ptr(k),
				Value: gitlab.Ptr(v),
			})
		}
		glOpts.Variables = &vars
	}

	_, resp, err := s.client.Pipelines.CreatePipeline(pid, glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabCIService) CancelRun(ctx context.Context, owner, repo string, runID int64) error {
	pid := owner + "/" + repo
	_, resp, err := s.client.Pipelines.CancelPipelineBuild(pid, runID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabCIService) RetryRun(ctx context.Context, owner, repo string, runID int64) error {
	pid := owner + "/" + repo
	_, resp, err := s.client.Pipelines.RetryPipelineBuild(pid, runID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabCIService) GetJobLog(ctx context.Context, owner, repo string, jobID int64) (io.ReadCloser, error) {
	pid := owner + "/" + repo
	reader, resp, err := s.client.Jobs.GetTraceFile(pid, jobID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	return io.NopCloser(reader), nil
}

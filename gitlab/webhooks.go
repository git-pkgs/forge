//go:build !tinygo

package gitlab

import (
	"context"
	"fmt"
	forge "github.com/git-pkgs/forge"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// gitLabWebhookEvents maps normalized event names onto GitLab's per-event
// hook flags, in the order they are reported by List and Get.
var gitLabWebhookEvents = []struct {
	name string
	get  func(*gitlab.ProjectHook) bool
	set  func(*gitlab.AddProjectHookOptions, *bool)
}{
	{"push", func(h *gitlab.ProjectHook) bool { return h.PushEvents }, func(o *gitlab.AddProjectHookOptions, v *bool) { o.PushEvents = v }},
	{"tag_push", func(h *gitlab.ProjectHook) bool { return h.TagPushEvents }, func(o *gitlab.AddProjectHookOptions, v *bool) { o.TagPushEvents = v }},
	{"issues", func(h *gitlab.ProjectHook) bool { return h.IssuesEvents }, func(o *gitlab.AddProjectHookOptions, v *bool) { o.IssuesEvents = v }},
	{"issue_comment", func(h *gitlab.ProjectHook) bool { return h.NoteEvents }, func(o *gitlab.AddProjectHookOptions, v *bool) { o.NoteEvents = v }},
	{"pull_request", func(h *gitlab.ProjectHook) bool { return h.MergeRequestsEvents }, func(o *gitlab.AddProjectHookOptions, v *bool) { o.MergeRequestsEvents = v }},
	{"release", func(h *gitlab.ProjectHook) bool { return h.ReleasesEvents }, func(o *gitlab.AddProjectHookOptions, v *bool) { o.ReleasesEvents = v }},
	{"deployment", func(h *gitlab.ProjectHook) bool { return h.DeploymentEvents }, func(o *gitlab.AddProjectHookOptions, v *bool) { o.DeploymentEvents = v }},
	{"gollum", func(h *gitlab.ProjectHook) bool { return h.WikiPageEvents }, func(o *gitlab.AddProjectHookOptions, v *bool) { o.WikiPageEvents = v }},
	{"workflow_run", func(h *gitlab.ProjectHook) bool { return h.PipelineEvents }, func(o *gitlab.AddProjectHookOptions, v *bool) { o.PipelineEvents = v }},
	{"workflow_job", func(h *gitlab.ProjectHook) bool { return h.JobEvents }, func(o *gitlab.AddProjectHookOptions, v *bool) { o.JobEvents = v }},
}

type gitLabWebhookService struct {
	client *gitlab.Client
}

func (f *gitLabForge) Webhooks() forge.WebhookService {
	return &gitLabWebhookService{client: f.client}
}

func convertGitLabHook(h *gitlab.ProjectHook) forge.Webhook {
	events := []string{}
	for _, e := range gitLabWebhookEvents {
		if e.get(h) {
			events = append(events, e.name)
		}
	}
	wh := forge.Webhook{
		ID:          h.ID,
		URL:         h.URL,
		Events:      events,
		Active:      h.AlertStatus != "disabled" && h.AlertStatus != "temporarily_disabled",
		ContentType: "json",
		InsecureSSL: !h.EnableSSLVerification,
	}
	if h.CreatedAt != nil {
		wh.CreatedAt = *h.CreatedAt
	}
	return wh
}

// gitLabHookOptions builds the add options for a webhook. Every known event
// flag is sent explicitly because GitLab enables push events by default.
func gitLabHookOptions(opts forge.CreateWebhookOpts) (*gitlab.AddProjectHookOptions, error) {
	events := opts.Events
	if len(events) == 0 {
		events = []string{"push"}
	}
	enabled := make(map[string]bool, len(events))
	for _, name := range events {
		known := false
		for _, e := range gitLabWebhookEvents {
			if e.name == name {
				known = true
				break
			}
		}
		if !known {
			return nil, fmt.Errorf("gitlab: unsupported webhook event %q", name)
		}
		enabled[name] = true
	}

	glOpts := &gitlab.AddProjectHookOptions{
		URL:                   gitlab.Ptr(opts.URL),
		EnableSSLVerification: gitlab.Ptr(!opts.InsecureSSL),
	}
	if opts.Secret != "" {
		glOpts.Token = gitlab.Ptr(opts.Secret)
	}
	for _, e := range gitLabWebhookEvents {
		e.set(glOpts, gitlab.Ptr(enabled[e.name]))
	}
	return glOpts, nil
}

func (s *gitLabWebhookService) List(ctx context.Context, owner, repo string, opts forge.ListWebhookOpts) ([]forge.Webhook, error) {
	pid := owner + "/" + repo
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	glOpts := &gitlab.ListProjectHooksOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage), Page: int64(page)},
	}

	var all []forge.Webhook
	for {
		hooks, resp, err := s.client.Projects.ListProjectHooks(pid, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, wrapErr(err)
		}
		for _, h := range hooks {
			all = append(all, convertGitLabHook(h))
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

func (s *gitLabWebhookService) Get(ctx context.Context, owner, repo string, id int64) (*forge.Webhook, error) {
	pid := owner + "/" + repo
	h, resp, err := s.client.Projects.GetProjectHook(pid, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, wrapErr(err)
	}

	wh := convertGitLabHook(h)
	return &wh, nil
}

func (s *gitLabWebhookService) Create(ctx context.Context, owner, repo string, opts forge.CreateWebhookOpts) (*forge.Webhook, error) {
	pid := owner + "/" + repo
	glOpts, err := gitLabHookOptions(opts)
	if err != nil {
		return nil, err
	}

	h, resp, err := s.client.Projects.AddProjectHook(pid, glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, wrapErr(err)
	}

	wh := convertGitLabHook(h)
	return &wh, nil
}

func (s *gitLabWebhookService) Delete(ctx context.Context, owner, repo string, id int64) error {
	pid := owner + "/" + repo
	resp, err := s.client.Projects.DeleteProjectHook(pid, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return wrapErr(err)
	}
	return nil
}

func (s *gitLabWebhookService) ParseEvent(r *http.Request, secret string) (*forge.WebhookEvent, error) {
	return ParseWebhook(r, secret)
}

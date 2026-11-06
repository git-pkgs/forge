package github

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"github.com/google/go-github/v82/github"
)

type gitHubWebhookService struct {
	client *github.Client
}

func (f *gitHubForge) Webhooks() forge.WebhookService {
	return &gitHubWebhookService{client: f.client}
}

func convertGitHubHook(h *github.Hook) forge.Webhook {
	cfg := h.GetConfig()
	return forge.Webhook{
		ID:          h.GetID(),
		URL:         cfg.GetURL(),
		Events:      h.Events,
		Active:      h.GetActive(),
		ContentType: cfg.GetContentType(),
		InsecureSSL: cfg.GetInsecureSSL() == "1",
		CreatedAt:   h.GetCreatedAt().Time,
		UpdatedAt:   h.GetUpdatedAt().Time,
	}
}

func (s *gitHubWebhookService) List(ctx context.Context, owner, repo string, opts forge.ListWebhookOpts) ([]forge.Webhook, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.ListOptions{PerPage: perPage, Page: page}

	var all []forge.Webhook
	for {
		hooks, resp, err := s.client.Repositories.ListHooks(ctx, owner, repo, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, wrapErr(err)
		}
		for _, h := range hooks {
			all = append(all, convertGitHubHook(h))
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

func (s *gitHubWebhookService) Get(ctx context.Context, owner, repo string, id int64) (*forge.Webhook, error) {
	h, resp, err := s.client.Repositories.GetHook(ctx, owner, repo, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, wrapErr(err)
	}

	result := convertGitHubHook(h)
	return &result, nil
}

func (s *gitHubWebhookService) Create(ctx context.Context, owner, repo string, opts forge.CreateWebhookOpts) (*forge.Webhook, error) {
	contentType := opts.ContentType
	if contentType == "" {
		contentType = "json"
	}
	insecureSSL := "0"
	if opts.InsecureSSL {
		insecureSSL = "1"
	}

	cfg := &github.HookConfig{
		URL:         github.Ptr(opts.URL),
		ContentType: github.Ptr(contentType),
		InsecureSSL: github.Ptr(insecureSSL),
	}
	if opts.Secret != "" {
		cfg.Secret = github.Ptr(opts.Secret)
	}

	h, resp, err := s.client.Repositories.CreateHook(ctx, owner, repo, &github.Hook{
		Config: cfg,
		Events: opts.Events,
		Active: github.Ptr(true),
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, wrapErr(err)
	}

	result := convertGitHubHook(h)
	return &result, nil
}

func (s *gitHubWebhookService) Delete(ctx context.Context, owner, repo string, id int64) error {
	resp, err := s.client.Repositories.DeleteHook(ctx, owner, repo, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return wrapErr(err)
	}
	return nil
}

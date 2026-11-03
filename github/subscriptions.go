package github

import (
	"context"
	"net/http"

	forge "github.com/git-pkgs/forge"
	"github.com/google/go-github/v82/github"
)

func convertGitHubSubscription(s *github.Subscription) *forge.Subscription {
	if s == nil {
		return &forge.Subscription{}
	}
	return &forge.Subscription{
		Subscribed: s.GetSubscribed(),
		Ignored:    s.GetIgnored(),
	}
}

func gitHubSubscriptionRequest(opts forge.SetSubscriptionOpts) *github.Subscription {
	return &github.Subscription{
		Subscribed: github.Ptr(!opts.Ignored),
		Ignored:    github.Ptr(opts.Ignored),
	}
}

// GetRepoSubscription reports an unwatched repository as an empty
// Subscription. GitHub answers 404 both for that and for a repository that
// does not exist, so the two cannot be told apart.
func (s *gitHubNotificationService) GetRepoSubscription(ctx context.Context, owner, repo string) (*forge.Subscription, error) {
	sub, _, err := s.client.Activity.GetRepositorySubscription(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	return convertGitHubSubscription(sub), nil
}

func (s *gitHubNotificationService) SetRepoSubscription(ctx context.Context, owner, repo string, opts forge.SetSubscriptionOpts) (*forge.Subscription, error) {
	sub, resp, err := s.client.Activity.SetRepositorySubscription(ctx, owner, repo, gitHubSubscriptionRequest(opts))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	return convertGitHubSubscription(sub), nil
}

func (s *gitHubNotificationService) DeleteRepoSubscription(ctx context.Context, owner, repo string) error {
	resp, err := s.client.Activity.DeleteRepositorySubscription(ctx, owner, repo)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitHubNotificationService) GetThreadSubscription(ctx context.Context, id string) (*forge.Subscription, error) {
	sub, resp, err := s.client.Activity.GetThreadSubscription(ctx, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	return convertGitHubSubscription(sub), nil
}

func (s *gitHubNotificationService) SetThreadSubscription(ctx context.Context, id string, opts forge.SetSubscriptionOpts) (*forge.Subscription, error) {
	sub, resp, err := s.client.Activity.SetThreadSubscription(ctx, id, gitHubSubscriptionRequest(opts))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	return convertGitHubSubscription(sub), nil
}

func (s *gitHubNotificationService) DeleteThreadSubscription(ctx context.Context, id string) error {
	resp, err := s.client.Activity.DeleteThreadSubscription(ctx, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

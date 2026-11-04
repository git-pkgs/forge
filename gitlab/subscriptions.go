//go:build !tinygo

package gitlab

import (
	"context"
	"fmt"
	"net/http"

	forge "github.com/git-pkgs/forge"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// GitLab has no repository watch endpoint, so repository subscriptions map to
// the project's notification level: "watch" is subscribed, "disabled" is
// ignored and deleting the subscription goes back to "global", the user's
// default. Other levels, such as "participating" or "custom", report neither.

func convertGitLabNotificationLevel(settings *gitlab.NotificationSettings) *forge.Subscription {
	if settings == nil {
		return &forge.Subscription{}
	}
	return &forge.Subscription{
		Subscribed: settings.Level == gitlab.WatchNotificationLevel,
		Ignored:    settings.Level == gitlab.DisabledNotificationLevel,
	}
}

func (s *gitLabNotificationService) setProjectNotificationLevel(ctx context.Context, owner, repo string, level gitlab.NotificationLevelValue) (*gitlab.NotificationSettings, error) {
	settings, resp, err := s.client.NotificationSettings.UpdateSettingsForProject(owner+"/"+repo, &gitlab.NotificationSettingsOptions{
		Level: gitlab.Ptr(level),
	}, gitlab.WithContext(ctx))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	return settings, nil
}

func (s *gitLabNotificationService) GetRepoSubscription(ctx context.Context, owner, repo string) (*forge.Subscription, error) {
	settings, resp, err := s.client.NotificationSettings.GetSettingsForProject(owner+"/"+repo, gitlab.WithContext(ctx))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	return convertGitLabNotificationLevel(settings), nil
}

func (s *gitLabNotificationService) SetRepoSubscription(ctx context.Context, owner, repo string, opts forge.SetSubscriptionOpts) (*forge.Subscription, error) {
	level := gitlab.WatchNotificationLevel
	if opts.Ignored {
		level = gitlab.DisabledNotificationLevel
	}
	settings, err := s.setProjectNotificationLevel(ctx, owner, repo, level)
	if err != nil {
		return nil, err
	}
	return convertGitLabNotificationLevel(settings), nil
}

func (s *gitLabNotificationService) DeleteRepoSubscription(ctx context.Context, owner, repo string) error {
	_, err := s.setProjectNotificationLevel(ctx, owner, repo, gitlab.GlobalNotificationLevel)
	return err
}

// GitLab notifications are todos, which cannot be subscribed to. GitLab
// subscriptions belong to issues and merge requests instead.

func (s *gitLabNotificationService) GetThreadSubscription(ctx context.Context, id string) (*forge.Subscription, error) {
	return nil, fmt.Errorf("getting thread subscription: %w", forge.ErrNotSupported)
}

func (s *gitLabNotificationService) SetThreadSubscription(ctx context.Context, id string, opts forge.SetSubscriptionOpts) (*forge.Subscription, error) {
	return nil, fmt.Errorf("setting thread subscription: %w", forge.ErrNotSupported)
}

func (s *gitLabNotificationService) DeleteThreadSubscription(ctx context.Context, id string) error {
	return fmt.Errorf("deleting thread subscription: %w", forge.ErrNotSupported)
}

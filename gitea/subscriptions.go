package gitea

import (
	"context"
	"fmt"

	forge "github.com/git-pkgs/forge"
)

// GetRepoSubscription reports an unwatched repository as an empty
// Subscription. Gitea answers 404 both for that and for a repository that
// does not exist, so the two cannot be told apart.
func (s *giteaNotificationService) GetRepoSubscription(ctx context.Context, owner, repo string) (*forge.Subscription, error) {
	watching, resp, err := s.client.CheckRepoWatch(owner, repo)
	if err != nil {
		return nil, wrapErr("get repository subscription", resp, err)
	}
	return &forge.Subscription{Subscribed: watching}, nil
}

// SetRepoSubscription watches the repository. Gitea has no way to ignore a
// repository, so opts.Ignored returns ErrNotSupported.
func (s *giteaNotificationService) SetRepoSubscription(ctx context.Context, owner, repo string, opts forge.SetSubscriptionOpts) (*forge.Subscription, error) {
	if opts.Ignored {
		return nil, fmt.Errorf("ignoring a repository: %w", forge.ErrNotSupported)
	}
	resp, err := s.client.WatchRepo(owner, repo)
	if err != nil {
		return nil, wrapErr("set repository subscription", resp, err)
	}
	return &forge.Subscription{Subscribed: true}, nil
}

func (s *giteaNotificationService) DeleteRepoSubscription(ctx context.Context, owner, repo string) error {
	resp, err := s.client.UnWatchRepo(owner, repo)
	if err != nil {
		return wrapErr("delete repository subscription", resp, err)
	}
	return nil
}

// Gitea has subscriptions on issues and pull requests but not on
// notification threads.

func (s *giteaNotificationService) GetThreadSubscription(ctx context.Context, id string) (*forge.Subscription, error) {
	return nil, fmt.Errorf("getting thread subscription: %w", forge.ErrNotSupported)
}

func (s *giteaNotificationService) SetThreadSubscription(ctx context.Context, id string, opts forge.SetSubscriptionOpts) (*forge.Subscription, error) {
	return nil, fmt.Errorf("setting thread subscription: %w", forge.ErrNotSupported)
}

func (s *giteaNotificationService) DeleteThreadSubscription(ctx context.Context, id string) error {
	return fmt.Errorf("deleting thread subscription: %w", forge.ErrNotSupported)
}

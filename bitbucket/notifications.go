package bitbucket

import (
	"context"
	"fmt"

	forge "github.com/git-pkgs/forge"
)

type bitbucketNotificationService struct{}

func (f *bitbucketForge) Notifications() forge.NotificationService {
	return &bitbucketNotificationService{}
}

func (s *bitbucketNotificationService) List(ctx context.Context, opts forge.ListNotificationOpts) ([]forge.Notification, error) {
	return nil, fmt.Errorf("listing notifications: %w", forge.ErrNotSupported)
}

func (s *bitbucketNotificationService) MarkRead(ctx context.Context, opts forge.MarkNotificationOpts) error {
	return fmt.Errorf("marking notifications: %w", forge.ErrNotSupported)
}

func (s *bitbucketNotificationService) Get(ctx context.Context, id string) (*forge.Notification, error) {
	return nil, fmt.Errorf("getting notification: %w", forge.ErrNotSupported)
}

func (s *bitbucketNotificationService) GetRepoSubscription(ctx context.Context, owner, repo string) (*forge.Subscription, error) {
	return nil, fmt.Errorf("getting repository subscription: %w", forge.ErrNotSupported)
}

func (s *bitbucketNotificationService) SetRepoSubscription(ctx context.Context, owner, repo string, opts forge.SetSubscriptionOpts) (*forge.Subscription, error) {
	return nil, fmt.Errorf("setting repository subscription: %w", forge.ErrNotSupported)
}

func (s *bitbucketNotificationService) DeleteRepoSubscription(ctx context.Context, owner, repo string) error {
	return fmt.Errorf("deleting repository subscription: %w", forge.ErrNotSupported)
}

func (s *bitbucketNotificationService) GetThreadSubscription(ctx context.Context, id string) (*forge.Subscription, error) {
	return nil, fmt.Errorf("getting thread subscription: %w", forge.ErrNotSupported)
}

func (s *bitbucketNotificationService) SetThreadSubscription(ctx context.Context, id string, opts forge.SetSubscriptionOpts) (*forge.Subscription, error) {
	return nil, fmt.Errorf("setting thread subscription: %w", forge.ErrNotSupported)
}

func (s *bitbucketNotificationService) DeleteThreadSubscription(ctx context.Context, id string) error {
	return fmt.Errorf("deleting thread subscription: %w", forge.ErrNotSupported)
}

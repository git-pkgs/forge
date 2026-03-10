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

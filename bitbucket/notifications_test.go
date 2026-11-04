package bitbucket

import (
	"context"
	"errors"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestBitbucketNotificationsNotSupported(t *testing.T) {
	f := New("test-token", nil)

	_, err := f.Notifications().List(context.Background(), forge.ListNotificationOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}

	err = f.Notifications().MarkRead(context.Background(), forge.MarkNotificationOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}

	_, err = f.Notifications().Get(context.Background(), "1")
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}
}

func TestBitbucketSubscriptionsNotSupported(t *testing.T) {
	ctx := context.Background()
	n := New("test-token", nil).Notifications()

	if _, err := n.GetRepoSubscription(ctx, "owner", "repo"); !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("GetRepoSubscription: expected ErrNotSupported, got %v", err)
	}
	if _, err := n.SetRepoSubscription(ctx, "owner", "repo", forge.SetSubscriptionOpts{}); !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("SetRepoSubscription: expected ErrNotSupported, got %v", err)
	}
	if err := n.DeleteRepoSubscription(ctx, "owner", "repo"); !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("DeleteRepoSubscription: expected ErrNotSupported, got %v", err)
	}
	if _, err := n.GetThreadSubscription(ctx, "1"); !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("GetThreadSubscription: expected ErrNotSupported, got %v", err)
	}
	if _, err := n.SetThreadSubscription(ctx, "1", forge.SetSubscriptionOpts{}); !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("SetThreadSubscription: expected ErrNotSupported, got %v", err)
	}
	if err := n.DeleteThreadSubscription(ctx, "1"); !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("DeleteThreadSubscription: expected ErrNotSupported, got %v", err)
	}
}

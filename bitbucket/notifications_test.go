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

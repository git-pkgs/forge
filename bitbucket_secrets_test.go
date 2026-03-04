package forges

import (
	"context"
	"errors"
	"testing"
)

func TestBitbucketSecretNotSupported(t *testing.T) {
	svc := &bitbucketSecretService{}

	_, err := svc.List(context.Background(), "owner", "repo", ListSecretOpts{})
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("List: expected ErrNotSupported, got %v", err)
	}

	err = svc.Set(context.Background(), "owner", "repo", SetSecretOpts{})
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("Set: expected ErrNotSupported, got %v", err)
	}

	err = svc.Delete(context.Background(), "owner", "repo", "MY_SECRET")
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("Delete: expected ErrNotSupported, got %v", err)
	}
}

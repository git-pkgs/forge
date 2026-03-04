package bitbucket

import (
	"context"
	"errors"
	forge "github.com/git-pkgs/forge"
	"testing"
)

func TestBitbucketSecretNotSupported(t *testing.T) {
	svc := &bitbucketSecretService{}

	_, err := svc.List(context.Background(), "owner", "repo", forge.ListSecretOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("List: expected forge.ErrNotSupported, got %v", err)
	}

	err = svc.Set(context.Background(), "owner", "repo", forge.SetSecretOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("Set: expected forge.ErrNotSupported, got %v", err)
	}

	err = svc.Delete(context.Background(), "owner", "repo", "MY_SECRET")
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("Delete: expected forge.ErrNotSupported, got %v", err)
	}
}

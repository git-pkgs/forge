package bitbucket

import (
	"context"
	"errors"
	forge "github.com/git-pkgs/forge"
	"testing"
)

func TestBitbucketDeployKeyNotSupported(t *testing.T) {
	svc := &bitbucketDeployKeyService{}

	_, err := svc.List(context.Background(), "owner", "repo", forge.ListDeployKeyOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("List: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = svc.Get(context.Background(), "owner", "repo", 1)
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("Get: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = svc.Create(context.Background(), "owner", "repo", forge.CreateDeployKeyOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("Create: expected forge.ErrNotSupported, got %v", err)
	}

	err = svc.Delete(context.Background(), "owner", "repo", 1)
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("Delete: expected forge.ErrNotSupported, got %v", err)
	}
}

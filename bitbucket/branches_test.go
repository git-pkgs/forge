package bitbucket

import (
	"context"
	"errors"
	forge "github.com/git-pkgs/forge"
	"testing"
)

func TestBitbucketBranchNotSupported(t *testing.T) {
	svc := &bitbucketBranchService{}

	_, err := svc.List(context.Background(), "owner", "repo", forge.ListBranchOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("List: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = svc.Create(context.Background(), "owner", "repo", "new", "main")
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("Create: expected forge.ErrNotSupported, got %v", err)
	}

	err = svc.Delete(context.Background(), "owner", "repo", "old")
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("Delete: expected forge.ErrNotSupported, got %v", err)
	}
}

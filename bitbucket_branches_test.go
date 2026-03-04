package forges

import (
	"context"
	"errors"
	"testing"
)

func TestBitbucketBranchNotSupported(t *testing.T) {
	svc := &bitbucketBranchService{}

	_, err := svc.List(context.Background(), "owner", "repo", ListBranchOpts{})
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("List: expected ErrNotSupported, got %v", err)
	}

	_, err = svc.Create(context.Background(), "owner", "repo", "new", "main")
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("Create: expected ErrNotSupported, got %v", err)
	}

	err = svc.Delete(context.Background(), "owner", "repo", "old")
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("Delete: expected ErrNotSupported, got %v", err)
	}
}

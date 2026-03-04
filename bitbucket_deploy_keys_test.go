package forges

import (
	"context"
	"errors"
	"testing"
)

func TestBitbucketDeployKeyNotSupported(t *testing.T) {
	svc := &bitbucketDeployKeyService{}

	_, err := svc.List(context.Background(), "owner", "repo", ListDeployKeyOpts{})
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("List: expected ErrNotSupported, got %v", err)
	}

	_, err = svc.Get(context.Background(), "owner", "repo", 1)
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("Get: expected ErrNotSupported, got %v", err)
	}

	_, err = svc.Create(context.Background(), "owner", "repo", CreateDeployKeyOpts{})
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("Create: expected ErrNotSupported, got %v", err)
	}

	err = svc.Delete(context.Background(), "owner", "repo", 1)
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("Delete: expected ErrNotSupported, got %v", err)
	}
}

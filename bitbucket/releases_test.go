package bitbucket

import (
	"context"
	"errors"
	forge "github.com/git-pkgs/forge"
	"testing"
)

func TestBitbucketReleaseNotSupported(t *testing.T) {
	svc := &bitbucketReleaseService{}

	_, err := svc.List(context.Background(), "owner", "repo", forge.ListReleaseOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("List: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = svc.Get(context.Background(), "owner", "repo", "v1.0.0")
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("Get: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = svc.GetLatest(context.Background(), "owner", "repo")
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("GetLatest: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = svc.Create(context.Background(), "owner", "repo", forge.CreateReleaseOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("Create: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = svc.Update(context.Background(), "owner", "repo", "v1.0.0", forge.UpdateReleaseOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("Update: expected forge.ErrNotSupported, got %v", err)
	}

	err = svc.Delete(context.Background(), "owner", "repo", "v1.0.0")
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("Delete: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = svc.UploadAsset(context.Background(), "owner", "repo", "v1.0.0", nil)
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("UploadAsset: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = svc.DownloadAsset(context.Background(), "owner", "repo", 1)
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("DownloadAsset: expected forge.ErrNotSupported, got %v", err)
	}
}

package forges

import (
	"context"
	"errors"
	"testing"
)

func TestBitbucketReleaseNotSupported(t *testing.T) {
	svc := &bitbucketReleaseService{}

	_, err := svc.List(context.Background(), "owner", "repo", ListReleaseOpts{})
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("List: expected ErrNotSupported, got %v", err)
	}

	_, err = svc.Get(context.Background(), "owner", "repo", "v1.0.0")
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("Get: expected ErrNotSupported, got %v", err)
	}

	_, err = svc.GetLatest(context.Background(), "owner", "repo")
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("GetLatest: expected ErrNotSupported, got %v", err)
	}

	_, err = svc.Create(context.Background(), "owner", "repo", CreateReleaseOpts{})
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("Create: expected ErrNotSupported, got %v", err)
	}

	_, err = svc.Update(context.Background(), "owner", "repo", "v1.0.0", UpdateReleaseOpts{})
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("Update: expected ErrNotSupported, got %v", err)
	}

	err = svc.Delete(context.Background(), "owner", "repo", "v1.0.0")
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("Delete: expected ErrNotSupported, got %v", err)
	}

	_, err = svc.UploadAsset(context.Background(), "owner", "repo", "v1.0.0", nil)
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("UploadAsset: expected ErrNotSupported, got %v", err)
	}

	_, err = svc.DownloadAsset(context.Background(), "owner", "repo", 1)
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("DownloadAsset: expected ErrNotSupported, got %v", err)
	}
}

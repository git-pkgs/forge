package bitbucket

import (
	"context"
	"errors"
	forge "github.com/git-pkgs/forge"
	"testing"
)

func TestBitbucketCINotSupported(t *testing.T) {
	svc := &bitbucketCIService{}

	_, err := svc.ListRuns(context.Background(), "owner", "repo", forge.ListCIRunOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("ListRuns: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = svc.GetRun(context.Background(), "owner", "repo", 1)
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("GetRun: expected forge.ErrNotSupported, got %v", err)
	}

	err = svc.TriggerRun(context.Background(), "owner", "repo", forge.TriggerCIRunOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("TriggerRun: expected forge.ErrNotSupported, got %v", err)
	}

	err = svc.CancelRun(context.Background(), "owner", "repo", 1)
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("CancelRun: expected forge.ErrNotSupported, got %v", err)
	}

	err = svc.RetryRun(context.Background(), "owner", "repo", 1)
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("RetryRun: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = svc.GetJobLog(context.Background(), "owner", "repo", 1)
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("GetJobLog: expected forge.ErrNotSupported, got %v", err)
	}
}

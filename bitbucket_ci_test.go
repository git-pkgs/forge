package forges

import (
	"context"
	"errors"
	"testing"
)

func TestBitbucketCINotSupported(t *testing.T) {
	svc := &bitbucketCIService{}

	_, err := svc.ListRuns(context.Background(), "owner", "repo", ListCIRunOpts{})
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("ListRuns: expected ErrNotSupported, got %v", err)
	}

	_, err = svc.GetRun(context.Background(), "owner", "repo", 1)
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("GetRun: expected ErrNotSupported, got %v", err)
	}

	err = svc.TriggerRun(context.Background(), "owner", "repo", TriggerCIRunOpts{})
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("TriggerRun: expected ErrNotSupported, got %v", err)
	}

	err = svc.CancelRun(context.Background(), "owner", "repo", 1)
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("CancelRun: expected ErrNotSupported, got %v", err)
	}

	err = svc.RetryRun(context.Background(), "owner", "repo", 1)
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("RetryRun: expected ErrNotSupported, got %v", err)
	}

	_, err = svc.GetJobLog(context.Background(), "owner", "repo", 1)
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("GetJobLog: expected ErrNotSupported, got %v", err)
	}
}

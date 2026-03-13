package bitbucket

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"testing"
)

func TestBitbucketListCommitStatusesNotSupported(t *testing.T) {
	s := New("", nil).CommitStatuses()
	_, err := s.List(context.Background(), "owner", "repo", "abc123")
	if err != forge.ErrNotSupported {
		t.Errorf("List: expected forge.ErrNotSupported, got %v", err)
	}
}

func TestBitbucketSetCommitStatusNotSupported(t *testing.T) {
	s := New("", nil).CommitStatuses()
	_, err := s.Set(context.Background(), "owner", "repo", "abc123", forge.SetCommitStatusOpts{})
	if err != forge.ErrNotSupported {
		t.Errorf("Set: expected forge.ErrNotSupported, got %v", err)
	}
}

package bitbucket

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"testing"
)

func TestBitbucketListContributorsNotSupported(t *testing.T) {
	s := New("", nil).Repos()
	_, err := s.ListContributors(context.Background(), "owner", "repo")
	if err != forge.ErrNotSupported {
		t.Errorf("expected forge.ErrNotSupported, got %v", err)
	}
}

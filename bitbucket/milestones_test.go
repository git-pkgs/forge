package bitbucket

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"testing"
)

func TestBitbucketMilestonesNotSupported(t *testing.T) {
	s := &bitbucketMilestoneService{}

	_, err := s.List(context.Background(), "owner", "repo", forge.ListMilestoneOpts{})
	if err != forge.ErrNotSupported {
		t.Errorf("List: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = s.Get(context.Background(), "owner", "repo", 1)
	if err != forge.ErrNotSupported {
		t.Errorf("Get: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = s.Create(context.Background(), "owner", "repo", forge.CreateMilestoneOpts{Title: "v1"})
	if err != forge.ErrNotSupported {
		t.Errorf("Create: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = s.Update(context.Background(), "owner", "repo", 1, forge.UpdateMilestoneOpts{})
	if err != forge.ErrNotSupported {
		t.Errorf("Update: expected forge.ErrNotSupported, got %v", err)
	}

	err = s.Close(context.Background(), "owner", "repo", 1)
	if err != forge.ErrNotSupported {
		t.Errorf("Close: expected forge.ErrNotSupported, got %v", err)
	}

	err = s.Reopen(context.Background(), "owner", "repo", 1)
	if err != forge.ErrNotSupported {
		t.Errorf("Reopen: expected forge.ErrNotSupported, got %v", err)
	}

	err = s.Delete(context.Background(), "owner", "repo", 1)
	if err != forge.ErrNotSupported {
		t.Errorf("Delete: expected forge.ErrNotSupported, got %v", err)
	}
}

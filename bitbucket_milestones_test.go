package forges

import (
	"context"
	"testing"
)

func TestBitbucketMilestonesNotSupported(t *testing.T) {
	s := &bitbucketMilestoneService{}

	_, err := s.List(context.Background(), "owner", "repo", ListMilestoneOpts{})
	if err != ErrNotSupported {
		t.Errorf("List: expected ErrNotSupported, got %v", err)
	}

	_, err = s.Get(context.Background(), "owner", "repo", 1)
	if err != ErrNotSupported {
		t.Errorf("Get: expected ErrNotSupported, got %v", err)
	}

	_, err = s.Create(context.Background(), "owner", "repo", CreateMilestoneOpts{Title: "v1"})
	if err != ErrNotSupported {
		t.Errorf("Create: expected ErrNotSupported, got %v", err)
	}

	_, err = s.Update(context.Background(), "owner", "repo", 1, UpdateMilestoneOpts{})
	if err != ErrNotSupported {
		t.Errorf("Update: expected ErrNotSupported, got %v", err)
	}

	err = s.Close(context.Background(), "owner", "repo", 1)
	if err != ErrNotSupported {
		t.Errorf("Close: expected ErrNotSupported, got %v", err)
	}

	err = s.Reopen(context.Background(), "owner", "repo", 1)
	if err != ErrNotSupported {
		t.Errorf("Reopen: expected ErrNotSupported, got %v", err)
	}

	err = s.Delete(context.Background(), "owner", "repo", 1)
	if err != ErrNotSupported {
		t.Errorf("Delete: expected ErrNotSupported, got %v", err)
	}
}

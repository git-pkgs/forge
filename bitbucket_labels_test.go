package forges

import (
	"context"
	"testing"
)

func TestBitbucketLabelsNotSupported(t *testing.T) {
	s := &bitbucketLabelService{}

	_, err := s.List(context.Background(), "owner", "repo", ListLabelOpts{})
	if err != ErrNotSupported {
		t.Errorf("List: expected ErrNotSupported, got %v", err)
	}

	_, err = s.Get(context.Background(), "owner", "repo", "bug")
	if err != ErrNotSupported {
		t.Errorf("Get: expected ErrNotSupported, got %v", err)
	}

	_, err = s.Create(context.Background(), "owner", "repo", CreateLabelOpts{Name: "bug"})
	if err != ErrNotSupported {
		t.Errorf("Create: expected ErrNotSupported, got %v", err)
	}

	_, err = s.Update(context.Background(), "owner", "repo", "bug", UpdateLabelOpts{})
	if err != ErrNotSupported {
		t.Errorf("Update: expected ErrNotSupported, got %v", err)
	}

	err = s.Delete(context.Background(), "owner", "repo", "bug")
	if err != ErrNotSupported {
		t.Errorf("Delete: expected ErrNotSupported, got %v", err)
	}
}

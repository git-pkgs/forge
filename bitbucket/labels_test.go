package bitbucket

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"testing"
)

func TestBitbucketLabelsNotSupported(t *testing.T) {
	s := &bitbucketLabelService{}

	_, err := s.List(context.Background(), "owner", "repo", forge.ListLabelOpts{})
	if err != forge.ErrNotSupported {
		t.Errorf("List: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = s.Get(context.Background(), "owner", "repo", "bug")
	if err != forge.ErrNotSupported {
		t.Errorf("Get: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = s.Create(context.Background(), "owner", "repo", forge.CreateLabelOpts{Name: "bug"})
	if err != forge.ErrNotSupported {
		t.Errorf("Create: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = s.Update(context.Background(), "owner", "repo", "bug", forge.UpdateLabelOpts{})
	if err != forge.ErrNotSupported {
		t.Errorf("Update: expected forge.ErrNotSupported, got %v", err)
	}

	err = s.Delete(context.Background(), "owner", "repo", "bug")
	if err != forge.ErrNotSupported {
		t.Errorf("Delete: expected forge.ErrNotSupported, got %v", err)
	}
}

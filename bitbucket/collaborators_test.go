package bitbucket

import (
	"context"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestBitbucketCollaboratorsNotSupported(t *testing.T) {
	f := New("token", nil)
	ctx := context.Background()

	_, err := f.Collaborators().List(ctx, "owner", "repo", forge.ListCollaboratorOpts{})
	if err != forge.ErrNotSupported {
		t.Errorf("List: expected forge.ErrNotSupported, got %v", err)
	}

	err = f.Collaborators().Add(ctx, "owner", "repo", "user", forge.AddCollaboratorOpts{})
	if err != forge.ErrNotSupported {
		t.Errorf("Add: expected forge.ErrNotSupported, got %v", err)
	}

	err = f.Collaborators().Remove(ctx, "owner", "repo", "user")
	if err != forge.ErrNotSupported {
		t.Errorf("Remove: expected forge.ErrNotSupported, got %v", err)
	}
}

package bitbucket

import (
	"context"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestBitbucketFilesNotSupported(t *testing.T) {
	f := New("token", nil)
	ctx := context.Background()

	_, err := f.Files().Get(ctx, "owner", "repo", "path", "main")
	if err != forge.ErrNotSupported {
		t.Errorf("Get: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = f.Files().List(ctx, "owner", "repo", "path", "main")
	if err != forge.ErrNotSupported {
		t.Errorf("List: expected forge.ErrNotSupported, got %v", err)
	}
}

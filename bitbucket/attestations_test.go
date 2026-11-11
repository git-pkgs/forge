package bitbucket

import (
	"context"
	"errors"
	"strings"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestBitbucketAttestationsNotSupported(t *testing.T) {
	svc := &bitbucketAttestationService{}

	_, err := svc.List(context.Background(), "owner", "repo", "sha256:"+strings.Repeat("a", 64), forge.ListAttestationOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("List: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = svc.Download(context.Background(), "owner", "repo", forge.Attestation{ID: "1"})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("Download: expected forge.ErrNotSupported, got %v", err)
	}
}

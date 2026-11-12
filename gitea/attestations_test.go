package gitea

import (
	"context"
	"errors"
	"strings"
	"testing"

	forge "github.com/git-pkgs/forge"
)

func TestGiteaAttestationsNotSupported(t *testing.T) {
	svc := &giteaAttestationService{}

	_, err := svc.List(context.Background(), "testorg", "testrepo", "sha256:"+strings.Repeat("a", 64), forge.ListAttestationOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("List: expected forge.ErrNotSupported, got %v", err)
	}

	_, err = svc.Download(context.Background(), "testorg", "testrepo", forge.Attestation{ID: "1"})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Errorf("Download: expected forge.ErrNotSupported, got %v", err)
	}
}

package gitea

import (
	"context"
	"encoding/json"

	forge "github.com/git-pkgs/forge"
)

// Gitea and Forgejo have no artifact attestation API.
type giteaAttestationService struct{}

func (f *giteaForge) Attestations() forge.AttestationService {
	return &giteaAttestationService{}
}

func (s *giteaAttestationService) List(_ context.Context, _, _, _ string, _ forge.ListAttestationOpts) ([]forge.Attestation, error) {
	return nil, forge.ErrNotSupported
}

func (s *giteaAttestationService) Download(_ context.Context, _, _ string, _ forge.Attestation) (json.RawMessage, error) {
	return nil, forge.ErrNotSupported
}

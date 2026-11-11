package bitbucket

import (
	"context"
	"encoding/json"

	forge "github.com/git-pkgs/forge"
)

// Bitbucket Cloud has no artifact attestation API.
type bitbucketAttestationService struct{}

func (f *bitbucketForge) Attestations() forge.AttestationService {
	return &bitbucketAttestationService{}
}

func (s *bitbucketAttestationService) List(_ context.Context, _, _, _ string, _ forge.ListAttestationOpts) ([]forge.Attestation, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketAttestationService) Download(_ context.Context, _, _ string, _ forge.Attestation) (json.RawMessage, error) {
	return nil, forge.ErrNotSupported
}

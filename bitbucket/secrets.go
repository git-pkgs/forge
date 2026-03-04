package bitbucket

import (
	"context"

	forge "github.com/git-pkgs/forge"
)

type bitbucketSecretService struct{}

func (f *bitbucketForge) Secrets() forge.SecretService {
	return &bitbucketSecretService{}
}

func (s *bitbucketSecretService) List(_ context.Context, _, _ string, _ forge.ListSecretOpts) ([]forge.Secret, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketSecretService) Set(_ context.Context, _, _ string, _ forge.SetSecretOpts) error {
	return forge.ErrNotSupported
}

func (s *bitbucketSecretService) Delete(_ context.Context, _, _, _ string) error {
	return forge.ErrNotSupported
}

package forges

import "context"

type bitbucketSecretService struct{}

func (f *bitbucketForge) Secrets() SecretService {
	return &bitbucketSecretService{}
}

func (s *bitbucketSecretService) List(_ context.Context, _, _ string, _ ListSecretOpts) ([]Secret, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketSecretService) Set(_ context.Context, _, _ string, _ SetSecretOpts) error {
	return ErrNotSupported
}

func (s *bitbucketSecretService) Delete(_ context.Context, _, _, _ string) error {
	return ErrNotSupported
}

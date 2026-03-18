package github

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"github.com/google/go-github/v82/github"
	"golang.org/x/crypto/nacl/box"
)

const naclKeySize = 32

type gitHubSecretService struct {
	client *github.Client
}

func (f *gitHubForge) Secrets() forge.SecretService {
	return &gitHubSecretService{client: f.client}
}

func (s *gitHubSecretService) List(ctx context.Context, owner, repo string, opts forge.ListSecretOpts) ([]forge.Secret, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.ListOptions{PerPage: perPage, Page: page}

	var all []forge.Secret
	for {
		secrets, resp, err := s.client.Actions.ListRepoSecrets(ctx, owner, repo, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, sec := range secrets.Secrets {
			all = append(all, forge.Secret{
				Name:      sec.Name,
				CreatedAt: sec.CreatedAt.Time,
				UpdatedAt: sec.UpdatedAt.Time,
			})
		}
		if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		ghOpts.Page = resp.NextPage
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *gitHubSecretService) Set(ctx context.Context, owner, repo string, opts forge.SetSecretOpts) error {
	// Get the repo's public key for encrypting secrets
	pubKey, resp, err := s.client.Actions.GetRepoPublicKey(ctx, owner, repo)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}

	encrypted, err := encryptSecret(pubKey.GetKey(), opts.Value)
	if err != nil {
		return fmt.Errorf("encrypting secret: %w", err)
	}

	_, err = s.client.Actions.CreateOrUpdateRepoSecret(ctx, owner, repo, &github.EncryptedSecret{
		Name:           opts.Name,
		KeyID:          pubKey.GetKeyID(),
		EncryptedValue: encrypted,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *gitHubSecretService) Delete(ctx context.Context, owner, repo, name string) error {
	resp, err := s.client.Actions.DeleteRepoSecret(ctx, owner, repo, name)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

// encryptSecret encrypts a secret value using the repo's NaCl public key.
func encryptSecret(publicKeyB64, secretValue string) (string, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return "", fmt.Errorf("decoding public key: %w", err)
	}

	var recipientKey [32]byte
	if len(publicKeyBytes) != naclKeySize {
		return "", fmt.Errorf("public key must be %d bytes, got %d", naclKeySize, len(publicKeyBytes))
	}
	copy(recipientKey[:], publicKeyBytes)

	encrypted, err := box.SealAnonymous(nil, []byte(secretValue), &recipientKey, rand.Reader)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

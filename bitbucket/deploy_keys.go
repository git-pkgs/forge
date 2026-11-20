package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"time"

	forge "github.com/git-pkgs/forge"
)

type bitbucketDeployKeyService struct {
	token      string
	httpClient *http.Client
}

func (f *bitbucketForge) DeployKeys() forge.DeployKeyService {
	return &bitbucketDeployKeyService{token: f.token, httpClient: f.httpClient}
}

type bbDeployKey struct {
	ID        int64  `json:"id"`
	Key       string `json:"key"`
	Label     string `json:"label"`
	CreatedOn string `json:"created_on"`
}

// convertBitbucketDeployKey maps a Bitbucket deploy key. Bitbucket deploy
// keys can only read, so ReadOnly is always true.
func convertBitbucketDeployKey(bb bbDeployKey) forge.DeployKey {
	result := forge.DeployKey{
		ID:       bb.ID,
		Title:    bb.Label,
		Key:      bb.Key,
		ReadOnly: true,
	}
	if t, err := time.Parse(time.RFC3339, bb.CreatedOn); err == nil {
		result.CreatedAt = t
	}
	return result
}

func (s *bitbucketDeployKeyService) repoService() *bitbucketRepoService {
	return &bitbucketRepoService{token: s.token, httpClient: s.httpClient}
}

func deployKeysURL(owner, repo string) string {
	return fmt.Sprintf("%s/repositories/%s/%s/deploy-keys", bitbucketAPI, owner, repo)
}

func (s *bitbucketDeployKeyService) List(ctx context.Context, owner, repo string, opts forge.ListDeployKeyOpts) ([]forge.DeployKey, error) {
	u := deployKeysURL(owner, repo) + "?" + pageQuery(opts.Page, opts.PerPage, opts.Limit)
	return listPages(ctx, s.repoService(), u, opts.Limit, convertBitbucketDeployKey)
}

func (s *bitbucketDeployKeyService) Get(ctx context.Context, owner, repo string, id int64) (*forge.DeployKey, error) {
	var bb bbDeployKey
	u := fmt.Sprintf("%s/%d", deployKeysURL(owner, repo), id)
	if err := s.repoService().doJSON(ctx, http.MethodGet, u, nil, &bb); err != nil {
		return nil, err
	}
	result := convertBitbucketDeployKey(bb)
	return &result, nil
}

// Create adds a deploy key. Bitbucket has no deploy keys with write access,
// so asking for one fails with forge.ErrNotSupported instead of silently
// creating a key that can only read.
func (s *bitbucketDeployKeyService) Create(ctx context.Context, owner, repo string, opts forge.CreateDeployKeyOpts) (*forge.DeployKey, error) {
	if !opts.ReadOnly {
		return nil, fmt.Errorf("creating a deploy key with write access: %w", forge.ErrNotSupported)
	}

	body := map[string]string{
		"key":   opts.Key,
		"label": opts.Title,
	}
	var bb bbDeployKey
	if err := s.repoService().doJSON(ctx, http.MethodPost, deployKeysURL(owner, repo), body, &bb); err != nil {
		return nil, err
	}
	result := convertBitbucketDeployKey(bb)
	return &result, nil
}

func (s *bitbucketDeployKeyService) Delete(ctx context.Context, owner, repo string, id int64) error {
	u := fmt.Sprintf("%s/%d", deployKeysURL(owner, repo), id)
	return s.repoService().doJSON(ctx, http.MethodDelete, u, nil, nil)
}

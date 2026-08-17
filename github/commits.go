package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	forge "github.com/git-pkgs/forge"
	gh "github.com/google/go-github/v82/github"
)

// DefaultAPIBaseURL is the public GitHub REST API base URL.
const DefaultAPIBaseURL = "https://api.github.com/"

// CommitResolver resolves GitHub branch, tag and abbreviated commit refs to
// full commit SHAs. Full 40-character hexadecimal SHAs are returned without a
// network request.
type CommitResolver struct {
	client *gh.Client
}

// NewCommitResolver creates a commit resolver for the public GitHub API. The
// token may be empty for unauthenticated requests. A nil HTTP client uses the
// default client selected by go-github.
func NewCommitResolver(token string, client *http.Client) *CommitResolver {
	resolver, _ := NewCommitResolverWithBase(DefaultAPIBaseURL, token, client)
	return resolver
}

// NewCommitResolverWithBase creates a commit resolver for an explicit GitHub
// API base URL. The URL must include the API path for GitHub Enterprise and is
// normalized to end in a slash.
func NewCommitResolverWithBase(baseURL, token string, client *http.Client) (*CommitResolver, error) {
	api := gh.NewClient(client)
	if token != "" {
		api = api.WithAuthToken(token)
	}

	base, err := url.Parse(strings.TrimRight(baseURL, "/") + "/")
	if err != nil {
		return nil, fmt.Errorf("parse GitHub API base URL: %w", err)
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return nil, fmt.Errorf("parse GitHub API base URL: unsupported scheme %q", base.Scheme)
	}
	if base.Host == "" {
		return nil, errors.New("parse GitHub API base URL: host is required")
	}
	api.BaseURL = base

	return &CommitResolver{client: api}, nil
}

// ResolveCommit returns the full commit SHA for ref in owner/repo. GitHub's
// commit endpoint dereferences both lightweight and annotated tags.
func (r *CommitResolver) ResolveCommit(ctx context.Context, owner, repo, ref string) (string, error) {
	if err := forge.ValidateCommitRef(owner, repo, ref); err != nil {
		return "", err
	}
	if forge.IsFullCommitSHA(ref) {
		return strings.ToLower(ref), nil
	}
	if r == nil || r.client == nil {
		return "", errors.New("resolve GitHub commit: resolver is nil")
	}
	return resolveCommit(ctx, r.client, owner, repo, ref)
}

type gitHubCommitService struct {
	client *gh.Client
}

func (f *gitHubForge) Commits() forge.CommitService {
	return &gitHubCommitService{client: f.client}
}

// ResolveCommit returns the full commit SHA for ref in owner/repo.
func (s *gitHubCommitService) ResolveCommit(ctx context.Context, owner, repo, ref string) (string, error) {
	if err := forge.ValidateCommitRef(owner, repo, ref); err != nil {
		return "", err
	}
	if forge.IsFullCommitSHA(ref) {
		return strings.ToLower(ref), nil
	}
	return resolveCommit(ctx, s.client, owner, repo, ref)
}

// resolveCommit asks the GitHub commit endpoint for the full SHA behind ref.
// Callers have already rejected empty arguments and short-circuited refs that
// are full SHAs already.
func resolveCommit(ctx context.Context, client *gh.Client, owner, repo, ref string) (string, error) {
	sha, response, err := client.Repositories.GetCommitSHA1(ctx, owner, repo, ref, "")
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return "", forge.CommitRefError(owner, repo, ref, forge.ErrNotFound)
		}
		return "", forge.CommitRefError(owner, repo, ref, err)
	}
	return forge.ResolvedCommitSHA(owner, repo, ref, sha)
}

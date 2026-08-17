package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	forge "github.com/git-pkgs/forge"
	gh "github.com/google/go-github/v82/github"
)

// DefaultAPIBaseURL is the public GitHub REST API base URL.
const DefaultAPIBaseURL = "https://api.github.com/"

const fullCommitSHALength = 40

// CommitResolver resolves GitHub branch, tag, and abbreviated commit refs to
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
		return nil, fmt.Errorf("parse GitHub API base URL: host is required")
	}
	api.BaseURL = base

	return &CommitResolver{client: api}, nil
}

// ResolveCommit returns the full commit SHA for ref in owner/repo. GitHub's
// commit endpoint dereferences both lightweight and annotated tags.
func (r *CommitResolver) ResolveCommit(ctx context.Context, owner, repo, ref string) (string, error) {
	if owner == "" || repo == "" || ref == "" {
		return "", fmt.Errorf("resolve GitHub commit: owner, repo, and ref are required")
	}
	if isFullCommitSHA(ref) {
		return strings.ToLower(ref), nil
	}
	if r == nil || r.client == nil {
		return "", fmt.Errorf("resolve GitHub commit: resolver is nil")
	}

	sha, response, err := r.client.Repositories.GetCommitSHA1(ctx, owner, repo, ref, "")
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("resolve %s/%s ref %q: %w", owner, repo, ref, forge.ErrNotFound)
		}
		return "", fmt.Errorf("resolve %s/%s ref %q: %w", owner, repo, ref, err)
	}
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return "", fmt.Errorf("resolve %s/%s ref %q: empty SHA in response", owner, repo, ref)
	}
	if !isFullCommitSHA(sha) {
		return "", fmt.Errorf("resolve %s/%s ref %q: invalid full SHA in response", owner, repo, ref)
	}
	return strings.ToLower(sha), nil
}

// isFullCommitSHA reports whether ref is a full-length hexadecimal SHA-1.
func isFullCommitSHA(ref string) bool {
	if len(ref) != fullCommitSHALength {
		return false
	}
	for _, char := range ref {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}

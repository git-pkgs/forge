package forges

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// FullCommitSHALength is the number of hexadecimal characters in a full SHA-1
// commit ID.
const FullCommitSHALength = 40

// ErrCommitRefRequired is returned when owner, repo or ref is empty.
var ErrCommitRefRequired = errors.New("resolve commit: owner, repo and ref are required")

// CommitService resolves repository refs to immutable commit SHAs. Callers use
// it to pin a mutable ref, such as an action reference or a package source
// revision, to the exact commit it currently points at without listing every
// branch or tag on the repository.
type CommitService interface {
	// ResolveCommit returns the full commit SHA that ref points at in
	// owner/repo. ref may be a branch name, a tag name, an abbreviated
	// commit SHA or a full commit SHA. Implementations return ErrNotFound
	// when the ref does not exist on the repository. They return
	// ErrNotSupported when the forge exposes no ref resolution endpoint.
	ResolveCommit(ctx context.Context, owner, repo, ref string) (string, error)
}

// IsFullCommitSHA reports whether ref is already a full-length hexadecimal
// SHA-1. Backends call it to skip a network round trip for refs that are
// immutable to begin with.
func IsFullCommitSHA(ref string) bool {
	if len(ref) != FullCommitSHALength {
		return false
	}
	for _, char := range ref {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}

// ValidateCommitRef rejects the empty arguments every CommitService
// implementation has to guard against before issuing a request.
func ValidateCommitRef(owner, repo, ref string) error {
	if owner == "" || repo == "" || ref == "" {
		return ErrCommitRefRequired
	}
	return nil
}

// CommitRefError wraps err with the repository and ref being resolved so every
// backend reports resolution failures the same way.
func CommitRefError(owner, repo, ref string, err error) error {
	return fmt.Errorf("resolve %s/%s ref %q: %w", owner, repo, ref, err)
}

// ResolvedCommitSHA normalizes a commit SHA a forge API returned for
// owner/repo ref. It trims surrounding whitespace, lowercases the value and
// rejects anything that is not a full hexadecimal SHA-1, so callers never
// receive a value they cannot pin against.
func ResolvedCommitSHA(owner, repo, ref, sha string) (string, error) {
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return "", CommitRefError(owner, repo, ref, errors.New("empty SHA in response"))
	}
	if !IsFullCommitSHA(sha) {
		return "", CommitRefError(owner, repo, ref, fmt.Errorf("invalid full SHA %q in response", sha))
	}
	return strings.ToLower(sha), nil
}

// ResolveCommit resolves ref to a full commit SHA for the repository at
// repoURL, routing to the forge registered for that URL's domain.
func (c *Client) ResolveCommit(ctx context.Context, repoURL, ref string) (string, error) {
	domain, owner, repo, err := ParseRepoURL(repoURL)
	if err != nil {
		return "", err
	}
	f, err := c.forgeFor(domain)
	if err != nil {
		return "", err
	}
	return f.Commits().ResolveCommit(ctx, owner, repo, ref)
}

package forges

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// sha256DigestLength is the number of hexadecimal characters in a SHA-256
// digest.
const sha256DigestLength = 64

// ErrInvalidDigest is returned when an artifact digest is not of the form
// "sha256:<64 hex digits>".
var ErrInvalidDigest = errors.New("artifact digest must be sha256:<64 hex digits>")

// AttestationService discovers and retrieves artifact attestations, such as
// SLSA provenance or SBOMs signed as Sigstore bundles, that a forge stores for
// an artifact digest. Fetching is kept separate from verification: callers
// verify the returned bundles themselves.
type AttestationService interface {
	// List returns every attestation the repository holds for the artifact
	// with the given digest, following pagination. digest has the form
	// "sha256:<hex>". An empty result with a nil error means the lookup
	// succeeded and the forge holds no attestations for the artifact; request
	// failures, including authentication and rate limit errors, are returned
	// as errors. Implementations return ErrNotFound when the repository does
	// not exist and ErrNotSupported when the forge has no attestation API.
	List(ctx context.Context, owner, repo, digest string, opts ListAttestationOpts) ([]Attestation, error)
	// Download returns the original Sigstore bundle JSON for an attestation
	// returned by List. When the listing already included the bundle it is
	// returned without a request.
	Download(ctx context.Context, owner, repo string, att Attestation) (json.RawMessage, error)
}

// Attestation holds normalized metadata about an artifact attestation.
type Attestation struct {
	// ID identifies the attestation within the repository. GitLab sets the
	// attestation IID; GitHub has no such ID and leaves it empty.
	ID            string `json:"id,omitempty"`
	SubjectDigest string `json:"subject_digest"` // sha256:<hex>
	// PredicateType is the in-toto predicate type URI. It is empty when the
	// forge only reports it inside the bundle, as GitHub does.
	PredicateType string `json:"predicate_type,omitempty"`
	Status        string `json:"status,omitempty"` // forge-native, e.g. GitLab's success or error
	// BundleURL is where Download fetches the bundle from. GitHub's is a
	// signed blob storage link that expires shortly after listing.
	BundleURL string          `json:"bundle_url,omitempty"`
	Bundle    json.RawMessage `json:"bundle,omitempty"` // set when the listing includes the bundle
	CreatedAt time.Time       `json:"created_at,omitzero"`
	ExpiresAt time.Time       `json:"expires_at,omitzero"`
}

// ListAttestationOpts holds options for listing attestations.
type ListAttestationOpts struct {
	// PredicateType keeps only attestations of this predicate type. It
	// accepts a full predicate type URI or the short names "provenance" and
	// "sbom".
	PredicateType string
	Limit         int // max total results; 0 = unlimited
}

// ParseSHA256Digest validates an artifact digest of the form "sha256:<hex>"
// and returns its hex value in lowercase. Any other form returns
// ErrInvalidDigest.
func ParseSHA256Digest(digest string) (string, error) {
	hex, ok := strings.CutPrefix(digest, "sha256:")
	if !ok || len(hex) != sha256DigestLength {
		return "", fmt.Errorf("%w: %q", ErrInvalidDigest, digest)
	}
	for _, char := range hex {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return "", fmt.Errorf("%w: %q", ErrInvalidDigest, digest)
		}
	}
	return strings.ToLower(hex), nil
}

// ListAttestations lists the attestations for the artifact with the given
// digest in the repository at repoURL, routing to the forge registered for
// that URL's domain.
func (c *Client) ListAttestations(ctx context.Context, repoURL, digest string, opts ListAttestationOpts) ([]Attestation, error) {
	domain, owner, repo, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}
	f, err := c.forgeFor(domain)
	if err != nil {
		return nil, err
	}
	return f.Attestations().List(ctx, owner, repo, digest, opts)
}

// DownloadAttestation returns the Sigstore bundle for an attestation that
// ListAttestations returned for the repository at repoURL.
func (c *Client) DownloadAttestation(ctx context.Context, repoURL string, att Attestation) (json.RawMessage, error) {
	domain, owner, repo, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}
	f, err := c.forgeFor(domain)
	if err != nil {
		return nil, err
	}
	return f.Attestations().Download(ctx, owner, repo, att)
}

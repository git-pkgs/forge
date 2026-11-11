package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	forge "github.com/git-pkgs/forge"
	"github.com/golang/snappy"
	"github.com/google/go-github/v82/github"
)

// maxBundleBytes caps an attestation bundle downloaded from blob storage,
// both as served and once decompressed.
var maxBundleBytes = 32 << 20

type gitHubAttestationService struct {
	client     *github.Client
	httpClient *http.Client
}

func (f *gitHubForge) Attestations() forge.AttestationService {
	return &gitHubAttestationService{client: f.client, httpClient: f.httpClient}
}

// gitHubAttestation is one entry of the list attestations response. GitHub
// now leaves bundle null and links to the bundle with bundle_url, which
// go-github's Attestation type does not have.
type gitHubAttestation struct {
	Bundle    json.RawMessage `json:"bundle"`
	BundleURL string          `json:"bundle_url"`
}

// List follows GitHub's cursor pagination, which go-github's
// ListAttestations cannot do because it only takes page numbers.
func (s *gitHubAttestationService) List(ctx context.Context, owner, repo, digest string, opts forge.ListAttestationOpts) ([]forge.Attestation, error) {
	hex, err := forge.ParseSHA256Digest(digest)
	if err != nil {
		return nil, err
	}
	subject := "sha256:" + hex

	perPage := defaultPageSize
	if opts.Limit > 0 && opts.Limit < perPage {
		perPage = opts.Limit
	}
	query := url.Values{}
	query.Set("per_page", strconv.Itoa(perPage))
	if opts.PredicateType != "" {
		query.Set("predicate_type", opts.PredicateType)
	}
	path := fmt.Sprintf("repos/%s/%s/attestations/%s", url.PathEscape(owner), url.PathEscape(repo), subject)

	var all []forge.Attestation
	for {
		req, err := s.client.NewRequest(http.MethodGet, path+"?"+query.Encode(), nil)
		if err != nil {
			return nil, err
		}
		var page struct {
			Attestations []gitHubAttestation `json:"attestations"`
		}
		resp, err := s.client.Do(ctx, req, &page)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, a := range page.Attestations {
			att := forge.Attestation{SubjectDigest: subject, BundleURL: a.BundleURL}
			if len(a.Bundle) > 0 && !bytes.Equal(a.Bundle, []byte("null")) {
				att.Bundle = a.Bundle
			}
			all = append(all, att)
		}
		if resp.After == "" || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		query.Set("after", resp.After)
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

// Download fetches the snappy-compressed bundle that bundle_url points to.
// The URL is a signed blob storage link, so the request goes out without the
// API token and the URL is kept out of returned errors.
func (s *gitHubAttestationService) Download(ctx context.Context, _, _ string, att forge.Attestation) (json.RawMessage, error) {
	if len(att.Bundle) > 0 {
		return att.Bundle, nil
	}
	if att.BundleURL == "" {
		return nil, errors.New("github: attestation has no bundle or bundle URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, att.BundleURL, nil)
	if err != nil {
		return nil, errors.New("github: invalid attestation bundle URL")
	}
	hc := s.httpClient
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, fmt.Errorf("github: downloading attestation bundle: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github: downloading attestation bundle: HTTP %d", resp.StatusCode)
	}
	compressed, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxBundleBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("github: downloading attestation bundle: %w", err)
	}
	if len(compressed) > maxBundleBytes {
		return nil, fmt.Errorf("github: attestation bundle exceeds %d bytes", maxBundleBytes)
	}

	size, err := snappy.DecodedLen(compressed)
	if err != nil {
		return nil, fmt.Errorf("github: decoding attestation bundle: %w", err)
	}
	if size > maxBundleBytes {
		return nil, fmt.Errorf("github: attestation bundle exceeds %d bytes", maxBundleBytes)
	}
	bundle, err := snappy.Decode(nil, compressed)
	if err != nil {
		return nil, fmt.Errorf("github: decoding attestation bundle: %w", err)
	}
	if !json.Valid(bundle) {
		return nil, errors.New("github: attestation bundle is not valid JSON")
	}
	return bundle, nil
}

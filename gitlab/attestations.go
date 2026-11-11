//go:build !tinygo

package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	forge "github.com/git-pkgs/forge"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabAttestationService struct {
	client *gitlab.Client
}

func (f *gitLabForge) Attestations() forge.AttestationService {
	return &gitLabAttestationService{client: f.client}
}

func convertGitLabAttestation(a *gitlab.Attestation, subject string) forge.Attestation {
	att := forge.Attestation{
		ID:            strconv.FormatInt(a.IID, 10),
		SubjectDigest: subject,
		PredicateType: a.PredicateType,
		Status:        a.Status,
		BundleURL:     a.DownloadURL,
	}
	if a.CreatedAt != nil {
		att.CreatedAt = *a.CreatedAt
	}
	if a.ExpireAt != nil {
		att.ExpiresAt = *a.ExpireAt
	}
	return att
}

// gitLabAttestationMatches applies the predicate type filter, which GitLab
// has no parameter for. The short names provenance and sbom match GitLab's
// predicate_kind, the same names GitHub's predicate_type parameter accepts.
func gitLabAttestationMatches(a *gitlab.Attestation, predicateType string) bool {
	return predicateType == "" || a.PredicateType == predicateType || a.PredicateKind == predicateType
}

// List returns ErrNotFound when the project does not exist, and also when the
// instance or project has the attestations API turned off, because GitLab
// answers 404 in both cases.
func (s *gitLabAttestationService) List(ctx context.Context, owner, repo, digest string, opts forge.ListAttestationOpts) ([]forge.Attestation, error) {
	hex, err := forge.ParseSHA256Digest(digest)
	if err != nil {
		return nil, err
	}
	subject := "sha256:" + hex
	pid := owner + "/" + repo

	reqOpts := []gitlab.RequestOptionFunc{gitlab.WithContext(ctx)}
	var all []forge.Attestation
	for {
		atts, resp, err := s.client.Attestations.ListAttestations(pid, hex, reqOpts...)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, a := range atts {
			if gitLabAttestationMatches(a, opts.PredicateType) {
				all = append(all, convertGitLabAttestation(a, subject))
			}
		}
		if opts.Limit > 0 && len(all) >= opts.Limit {
			break
		}
		next, ok := gitlab.WithNext(resp)
		if !ok {
			break
		}
		reqOpts = []gitlab.RequestOptionFunc{gitlab.WithContext(ctx), next}
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *gitLabAttestationService) Download(ctx context.Context, owner, repo string, att forge.Attestation) (json.RawMessage, error) {
	if len(att.Bundle) > 0 {
		return att.Bundle, nil
	}
	iid, err := strconv.ParseInt(att.ID, 10, 64)
	if err != nil || iid <= 0 {
		return nil, fmt.Errorf("gitlab: invalid attestation ID %q", att.ID)
	}

	bundle, resp, err := s.client.Attestations.DownloadAttestation(owner+"/"+repo, iid, gitlab.WithContext(ctx))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	if !json.Valid(bundle) {
		return nil, errors.New("gitlab: attestation bundle is not valid JSON")
	}
	return bundle, nil
}

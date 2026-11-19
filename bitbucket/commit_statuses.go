package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	forge "github.com/git-pkgs/forge"
)

type bitbucketCommitStatusService struct {
	token      string
	httpClient *http.Client
}

func (f *bitbucketForge) CommitStatuses() forge.CommitStatusService {
	return &bitbucketCommitStatusService{token: f.token, httpClient: f.httpClient}
}

// bbCommitStatus is a Bitbucket build status. Key identifies the status on
// its commit, and setting a status with an existing key replaces it, so it
// plays the part of the context on other forges.
type bbCommitStatus struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	State       string `json:"state"` // SUCCESSFUL, FAILED, INPROGRESS, STOPPED
	Description string `json:"description"`
	URL         string `json:"url"`
	CreatedOn   string `json:"created_on"`
}

type bbCommitStatusesResponse struct {
	Values []bbCommitStatus `json:"values"`
	Next   string           `json:"next"`
}

func convertBitbucketCommitStatus(bb bbCommitStatus) forge.CommitStatus {
	result := forge.CommitStatus{
		State:       forge.NormalizeCommitStatusState(bb.State),
		Context:     bb.Key,
		Description: bb.Description,
		TargetURL:   bb.URL,
	}
	if t, err := time.Parse(time.RFC3339, bb.CreatedOn); err == nil {
		result.CreatedAt = t
	}
	return result
}

// bitbucketCommitStatusState maps a commit status state, normalized or
// Bitbucket's own, to the one Bitbucket takes. Bitbucket has no separate
// error state, so errors are reported as FAILED. A state with no Bitbucket
// equivalent, such as skipped, is sent upper-cased for Bitbucket to reject
// with its own validation error.
func bitbucketCommitStatusState(state string) string {
	switch forge.NormalizeCommitStatusState(state) {
	case forge.CommitStatusSuccess:
		return "SUCCESSFUL"
	case forge.CommitStatusFailure, forge.CommitStatusError:
		return "FAILED"
	case forge.CommitStatusPending:
		return "INPROGRESS"
	case forge.CommitStatusCancelled:
		return "STOPPED"
	default:
		return strings.ToUpper(state)
	}
}

func (s *bitbucketCommitStatusService) doJSON(ctx context.Context, method, url string, body any, v any) error {
	rs := &bitbucketRepoService{token: s.token, httpClient: s.httpClient}
	return rs.doJSON(ctx, method, url, body, v)
}

func commitStatusesURL(owner, repo, sha string) string {
	return fmt.Sprintf("%s/repositories/%s/%s/commit/%s/statuses", bitbucketAPI, owner, repo, url.PathEscape(sha))
}

func (s *bitbucketCommitStatusService) List(ctx context.Context, owner, repo, sha string) ([]forge.CommitStatus, error) {
	u := commitStatusesURL(owner, repo, sha) + "?pagelen=100"
	var all []forge.CommitStatus
	for u != "" {
		var page bbCommitStatusesResponse
		if err := s.doJSON(ctx, http.MethodGet, u, nil, &page); err != nil {
			return nil, err
		}
		for _, bb := range page.Values {
			all = append(all, convertBitbucketCommitStatus(bb))
		}
		u = page.Next
	}
	return all, nil
}

// Set creates a build status on the commit, or replaces the one with the
// same context. The context is used as both the key and the name Bitbucket
// shows for the status.
func (s *bitbucketCommitStatusService) Set(ctx context.Context, owner, repo, sha string, opts forge.SetCommitStatusOpts) (*forge.CommitStatus, error) {
	body := map[string]any{
		"key":   opts.Context,
		"name":  opts.Context,
		"state": bitbucketCommitStatusState(opts.State),
	}
	if opts.Description != "" {
		body["description"] = opts.Description
	}
	if opts.TargetURL != "" {
		body["url"] = opts.TargetURL
	}

	var bb bbCommitStatus
	if err := s.doJSON(ctx, http.MethodPost, commitStatusesURL(owner, repo, sha)+"/build", body, &bb); err != nil {
		return nil, err
	}
	result := convertBitbucketCommitStatus(bb)
	return &result, nil
}

// Package radicle implements a read-only forge backend for Radicle nodes
// running radicle-httpd.
//
// Radicle repositories have no owner. They are addressed by their repository
// ID (RID), such as rad:z3gqcJUoA1n9HaHKufZs5FCSGazv5, which callers pass as
// repo with an empty owner.
package radicle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	forges "github.com/git-pkgs/forge"
)

// RIDPrefix is the prefix every Radicle repository ID starts with.
const RIDPrefix = "rad:"

const maxErrorBodyBytes = 1 << 16

// maxResponseBytes caps how much of a successful response is read, since the
// node can be any seed the user points at. Blob JSON and raw files carry
// whole file contents, so the cap is sized for large files. It is a variable
// so tests can lower it.
var maxResponseBytes int64 = 32 << 20

type radicleForge struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a Radicle forge backend. The token is ignored because every endpoint it uses is public.
func New(baseURL, _ string, hc *http.Client) forges.Forge {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &radicleForge{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: hc,
	}
}

func (f *radicleForge) Repos() forges.RepoService               { return &repoService{f: f} }
func (f *radicleForge) Branches() forges.BranchService          { return &branchService{f: f} }
func (f *radicleForge) Files() forges.FileService               { return &fileService{f: f} }
func (f *radicleForge) Issues() forges.IssueService             { return unsupportedIssueService{} }
func (f *radicleForge) PullRequests() forges.PullRequestService { return unsupportedPRService{} }
func (f *radicleForge) Labels() forges.LabelService             { return unsupportedLabelService{} }
func (f *radicleForge) Milestones() forges.MilestoneService     { return unsupportedMilestoneService{} }
func (f *radicleForge) Releases() forges.ReleaseService         { return unsupportedReleaseService{} }
func (f *radicleForge) CI() forges.CIService                    { return unsupportedCIService{} }
func (f *radicleForge) DeployKeys() forges.DeployKeyService     { return unsupportedDeployKeyService{} }
func (f *radicleForge) Webhooks() forges.WebhookService         { return unsupportedWebhookService{} }
func (f *radicleForge) Secrets() forges.SecretService           { return unsupportedSecretService{} }
func (f *radicleForge) Notifications() forges.NotificationService {
	return unsupportedNotificationService{}
}
func (f *radicleForge) Reviews() forges.ReviewService { return unsupportedReviewService{} }
func (f *radicleForge) Collaborators() forges.CollaboratorService {
	return unsupportedCollaboratorService{}
}
func (f *radicleForge) CommitStatuses() forges.CommitStatusService {
	return unsupportedCommitStatusService{}
}
func (f *radicleForge) CheckRuns() forges.CheckRunService { return unsupportedCheckRunService{} }
func (f *radicleForge) Commits() forges.CommitService     { return unsupportedCommitService{} }

func (f *radicleForge) GetRateLimit(context.Context) (*forges.RateLimit, error) {
	return nil, forges.ErrNotSupported
}

func (f *radicleForge) GetAuthenticatedUser(context.Context) (*forges.User, error) {
	return nil, forges.ErrNotSupported
}

func (f *radicleForge) APIBaseURL() string {
	return f.baseURL + "/api/v1"
}

// rid validates owner and repo and returns the repository ID. Radicle has no
// owners, so a non-empty owner is rejected rather than silently ignored. The
// rad: prefix is optional on input.
func rid(owner, repo string) (string, error) {
	if owner != "" {
		return "", fmt.Errorf("radicle repositories have no owner, got %q (use -R rad:<RID>)", owner)
	}
	id := strings.TrimPrefix(repo, RIDPrefix)
	if id == "" || strings.ContainsAny(id, "/?#%") {
		return "", fmt.Errorf("invalid radicle repository ID %q", repo)
	}
	return RIDPrefix + id, nil
}

// repoPath returns the API path for a repository, with any extra path
// segments escaped individually. Callers pass segments from pathSegments,
// which has already resolved "." and "..", so escaping is enough to keep each
// one inside its own path element.
func (f *radicleForge) repoPath(id string, segments ...string) string {
	var b strings.Builder
	b.WriteString(f.APIBaseURL())
	b.WriteString("/repos/")
	b.WriteString(url.PathEscape(id))
	for _, s := range segments {
		b.WriteString("/")
		b.WriteString(url.PathEscape(s))
	}
	return b.String()
}

// get issues a GET request and returns the response body for a successful
// status. The caller closes the body.
func (f *radicleForge) get(ctx context.Context, u, accept string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()
		return nil, forges.ErrNotFound
	}
	// Anything outside 2xx is an error. That includes a 3xx the client did
	// not follow (no Location header), whose body must not be returned as
	// JSON or file content.
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		_ = resp.Body.Close()
		return nil, &forges.HTTPError{StatusCode: resp.StatusCode, URL: u, Body: string(body)}
	}
	return resp.Body, nil
}

func (f *radicleForge) getJSON(ctx context.Context, u string, v any) error {
	data, err := f.getRaw(ctx, u, "application/json")
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (f *radicleForge) getRaw(ctx context.Context, u, accept string) ([]byte, error) {
	body, err := f.get(ctx, u, accept)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()
	return readLimited(body, u, maxResponseBytes)
}

// readLimited reads at most limit bytes from r and fails if there is more.
func readLimited(r io.Reader, u string, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("radicle response from %s exceeds %d bytes", u, limit)
	}
	return data, nil
}

// rawPath returns the URL radicle-httpd serves a file's raw bytes at. Unlike
// the JSON API, this lives outside /api/v1.
func (f *radicleForge) rawPath(id, sha string, segments []string) string {
	parts := make([]string, 0, len(segments)+2)
	parts = append(parts, url.PathEscape(id), url.PathEscape(sha))
	for _, s := range segments {
		parts = append(parts, url.PathEscape(s))
	}
	return f.baseURL + "/raw/" + strings.Join(parts, "/")
}

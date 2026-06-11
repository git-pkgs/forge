package tangled

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	forges "github.com/git-pkgs/forge"
)

const (
	xrpcListBranches = "sh.tangled.git.temp.listBranches"
	xrpcListTags     = "sh.tangled.git.temp.listTags"
	xrpcGetTree      = "sh.tangled.git.temp.getTree"
)

type tangledForge struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// New creates a Tangled forge backend.
func New(baseURL, token string, hc *http.Client) forges.Forge {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &tangledForge{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: hc,
	}
}

func (f *tangledForge) Repos() forges.RepoService               { return &repoService{f: f} }
func (f *tangledForge) Branches() forges.BranchService          { return &branchService{f: f} }
func (f *tangledForge) Files() forges.FileService               { return &fileService{f: f} }
func (f *tangledForge) Issues() forges.IssueService             { return unsupportedIssueService{} }
func (f *tangledForge) PullRequests() forges.PullRequestService { return unsupportedPRService{} }
func (f *tangledForge) Labels() forges.LabelService             { return unsupportedLabelService{} }
func (f *tangledForge) Milestones() forges.MilestoneService     { return unsupportedMilestoneService{} }
func (f *tangledForge) Releases() forges.ReleaseService         { return unsupportedReleaseService{} }
func (f *tangledForge) CI() forges.CIService                    { return unsupportedCIService{} }
func (f *tangledForge) DeployKeys() forges.DeployKeyService     { return unsupportedDeployKeyService{} }
func (f *tangledForge) Secrets() forges.SecretService           { return unsupportedSecretService{} }
func (f *tangledForge) Notifications() forges.NotificationService {
	return unsupportedNotificationService{}
}
func (f *tangledForge) Reviews() forges.ReviewService { return unsupportedReviewService{} }
func (f *tangledForge) Collaborators() forges.CollaboratorService {
	return unsupportedCollaboratorService{}
}
func (f *tangledForge) CommitStatuses() forges.CommitStatusService {
	return unsupportedCommitStatusService{}
}

func (f *tangledForge) GetRateLimit(context.Context) (*forges.RateLimit, error) {
	return nil, forges.ErrNotSupported
}

func (f *tangledForge) xrpcURL(nsid string, params url.Values) string {
	u := f.baseURL + "/xrpc/" + nsid
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	return u
}

func (f *tangledForge) doJSON(ctx context.Context, method, url string, body any, v any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	if f.token != "" {
		req.Header.Set("Authorization", "Bearer "+f.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return forges.ErrNotFound
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		return &forges.HTTPError{StatusCode: resp.StatusCode, URL: url, Body: string(respBody)}
	}
	if v == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func (f *tangledForge) xrpc(ctx context.Context, nsid string, params url.Values, v any) error {
	return f.doJSON(ctx, http.MethodGet, f.xrpcURL(nsid, params), nil, v)
}

func (f *tangledForge) repoURL(owner, repo string) string {
	return f.baseURL + "/" + strings.Trim(path.Join(owner, repo), "/")
}

type repoMeta struct {
	DID         string
	CloneURL    string
	Description string
}

func (f *tangledForge) repoMeta(ctx context.Context, owner, repo string) (*repoMeta, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.repoURL(owner, repo), nil)
	if err != nil {
		return nil, err
	}
	if f.token != "" {
		req.Header.Set("Authorization", "Bearer "+f.token)
	}
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, forges.ErrNotFound
	}
	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return nil, &forges.HTTPError{StatusCode: resp.StatusCode, URL: req.URL.String(), Body: string(body)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseRepoMeta(string(body)), nil
}

func (f *tangledForge) repoDID(ctx context.Context, owner, repo string) (string, error) {
	if strings.HasPrefix(owner, "did:") {
		return owner, nil
	}
	meta, err := f.repoMeta(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	if meta.DID == "" {
		return "", fmt.Errorf("tangled repository DID not found for %s/%s", owner, repo)
	}
	return meta.DID, nil
}

var (
	metaTagRE     = regexp.MustCompile(`(?is)<meta\s+[^>]*>`)
	attrRE        = regexp.MustCompile(`(?is)\b([a-z0-9:_-]+)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	tangledRepoRE = regexp.MustCompile(`at://([^/]+)/sh\.tangled\.repo/[^"'\s>]+`)
)

func parseRepoMeta(body string) *repoMeta {
	meta := &repoMeta{}
	for _, tag := range metaTagRE.FindAllString(body, -1) {
		attrs := attrsFromTag(tag)
		name := strings.ToLower(attrs["name"])
		property := strings.ToLower(attrs["property"])
		content := html.UnescapeString(attrs["content"])
		switch {
		case name == "vcs:clone":
			meta.CloneURL = content
		case name == "description" || property == "og:description":
			meta.Description = content
		}
	}
	if match := tangledRepoRE.FindStringSubmatch(body); len(match) == 2 {
		meta.DID = html.UnescapeString(match[1])
	}
	return meta
}

func attrsFromTag(tag string) map[string]string {
	attrs := make(map[string]string)
	for _, match := range attrRE.FindAllStringSubmatch(tag, -1) {
		value := match[2]
		if value == "" {
			value = match[3]
		}
		attrs[strings.ToLower(match[1])] = value
	}
	return attrs
}

func perPage(limit, perPage int) int {
	if perPage > 0 {
		return perPage
	}
	if limit > 0 && limit < 100 {
		return limit
	}
	return 100
}

func limitReached(n, limit int) bool {
	return limit > 0 && n >= limit
}

func addLimit(params url.Values, limit int) {
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
}

func collection(raw any, names ...string) ([]any, string) {
	switch v := raw.(type) {
	case []any:
		return v, ""
	case map[string]any:
		cursor := stringField(v, "cursor", "nextCursor", "next_cursor", "next")
		for _, name := range names {
			if items, ok := v[name]; ok {
				switch vv := items.(type) {
				case []any:
					return vv, cursor
				case map[string]any:
					out := make([]any, 0, len(vv))
					for key, value := range vv {
						if m, ok := value.(map[string]any); ok {
							m["name"] = key
							out = append(out, m)
						}
					}
					return out, cursor
				}
			}
		}
	}
	return nil, ""
}

func stringField(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			switch vv := v.(type) {
			case string:
				return vv
			case map[string]any:
				if s := stringField(vv, "sha", "oid", "id", "hash"); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func boolField(m map[string]any, keys ...string) bool {
	for _, key := range keys {
		if v, ok := m[key].(bool); ok {
			return v
		}
	}
	return false
}

func int64Field(m map[string]any, keys ...string) int64 {
	for _, key := range keys {
		switch v := m[key].(type) {
		case float64:
			return int64(v)
		case int64:
			return v
		case json.Number:
			n, _ := v.Int64()
			return n
		}
	}
	return 0
}

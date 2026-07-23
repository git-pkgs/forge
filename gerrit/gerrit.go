package gerrit

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	forge "github.com/git-pkgs/forge"
)

const (
	defaultPageSize = 100
	stateAll        = "all"
	stateClosed     = "closed"
	stateMerged     = "merged"
	stateOpen       = "open"
)

type gerritForge struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// New creates a Gerrit forge backend.
func New(baseURL, token string, hc *http.Client) forge.Forge {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &gerritForge{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: hc,
	}
}

func (f *gerritForge) endpoint(apiPath string, values url.Values) string {
	prefix := ""
	if f.token != "" {
		prefix = "/a"
	}
	u := f.baseURL + prefix + apiPath
	if len(values) > 0 {
		u += "?" + values.Encode()
	}
	return u
}

func (f *gerritForge) APIBaseURL() string {
	if f.token != "" {
		return f.baseURL + "/a"
	}
	return f.baseURL
}

func (f *gerritForge) doJSON(ctx context.Context, method, apiPath string, query url.Values, body any, v any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, f.endpoint(apiPath, query), bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	f.authorize(req)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return forge.ErrNotFound
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		return &forge.HTTPError{StatusCode: resp.StatusCode, URL: req.URL.String(), Body: string(respBody)}
	}
	if v == nil {
		return nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(stripXSSI(respBody), v)
}

func (f *gerritForge) doText(ctx context.Context, method, apiPath string, query url.Values, body any) (string, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return "", err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, f.endpoint(apiPath, query), bodyReader)
	if err != nil {
		return "", err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	f.authorize(req)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", forge.ErrNotFound
	}
	if resp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(resp.Body)
		return "", &forge.HTTPError{StatusCode: resp.StatusCode, URL: req.URL.String(), Body: string(respBody)}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(stripXSSI(respBody)), nil
}

func (f *gerritForge) authorize(req *http.Request) {
	if f.token == "" {
		return
	}
	if user, password, ok := strings.Cut(f.token, ":"); ok {
		req.SetBasicAuth(user, password)
		return
	}
	req.Header.Set("Authorization", "Bearer "+f.token)
}

func stripXSSI(body []byte) []byte {
	return bytes.TrimPrefix(body, []byte(")]}'\n"))
}

func encodeID(id string) string {
	return url.PathEscape(id)
}

func projectName(owner, repo string) string {
	if owner == "" {
		return repo
	}
	return owner + "/" + repo
}

func splitProjectName(project string) (owner, repo string) {
	project = strings.Trim(project, "/")
	if project == "" {
		return "", ""
	}
	owner, repo = path.Split(project)
	return strings.TrimSuffix(owner, "/"), repo
}

func trimRefPrefix(ref string) string {
	for _, prefix := range []string{"refs/heads/", "refs/tags/"} {
		if strings.HasPrefix(ref, prefix) {
			return strings.TrimPrefix(ref, prefix)
		}
	}
	return ref
}

func parseGerritTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	} {
		if t, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func decodeBase64Text(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return string(decoded), nil
	}
	decoded, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func sortedMapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

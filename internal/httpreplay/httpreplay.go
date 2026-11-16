// Package httpreplay records real forge API traffic to fixture files and
// replays it in tests.
//
// Tests call Client to get an *http.Client for the backend under test. By
// default the client serves responses from the test's fixture file and never
// touches the network. With FORGE_RECORD=1 set, requests go to the real API
// and the exchanges are written to the fixture file when the test passes:
//
//	FORGE_RECORD=1 go test ./github -run TestReplay
//
// Fixtures live in testdata/replay/<test name>.json next to the test. Only
// the method, URL and body of each request are stored, so credentials sent
// in headers never reach a fixture. Of the response headers only the ones the
// SDKs read are kept, see keepHeader.
package httpreplay

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

const recordEnv = "FORGE_RECORD"

type interaction struct {
	Request  recordedRequest  `json:"request"`
	Response recordedResponse `json:"response"`
}

// recordedRequest holds the parts of a request used to match it during
// replay. Body is stored as text when it is valid UTF-8 and as base64
// otherwise, with BodyEncoding set to "base64".
type recordedRequest struct {
	Method       string `json:"method"`
	URL          string `json:"url"`
	Body         string `json:"body,omitempty"`
	BodyEncoding string `json:"body_encoding,omitempty"`
}

// recordedResponse holds a recorded response. Body is stored as text when it
// is valid UTF-8 and as base64 otherwise, with BodyEncoding set to "base64".
type recordedResponse struct {
	StatusCode   int         `json:"status_code"`
	Header       http.Header `json:"header,omitempty"`
	Body         string      `json:"body,omitempty"`
	BodyEncoding string      `json:"body_encoding,omitempty"`
}

// Client returns an HTTP client that replays the fixture for t, or records a
// new one when FORGE_RECORD=1 is set. A missing fixture, a request with no
// recorded match and a recorded request that is never made all fail t.
//
// Call Client before registering any cleanup that can fail t. Cleanups run in
// reverse order, so a cleanup registered earlier runs after the recorder has
// already written the fixture.
func Client(t testing.TB) *http.Client {
	t.Helper()
	path := fixturePath(t)
	if os.Getenv(recordEnv) == "1" {
		return &http.Client{Transport: newRecorder(t, path, http.DefaultTransport)}
	}
	r, err := newReplayer(t, path)
	if err != nil {
		t.Fatalf("httpreplay: %v; run with %s=1 to record it", err, recordEnv)
	}
	return &http.Client{Transport: r}
}

// fixturePath returns the fixture file used for t. Subtests map to nested
// directories.
func fixturePath(t testing.TB) string {
	return filepath.Join("testdata", "replay", filepath.FromSlash(t.Name())+".json")
}

// recorder sends requests to the network and keeps every exchange. The
// exchanges are written to path when the test finishes without failing.
type recorder struct {
	t    testing.TB
	base http.RoundTripper

	mu           sync.Mutex
	interactions []interaction
}

func newRecorder(t testing.TB, path string, base http.RoundTripper) *recorder {
	r := &recorder{t: t, base: base, interactions: []interaction{}}
	t.Cleanup(func() {
		if t.Failed() || t.Skipped() {
			t.Logf("httpreplay: test failed or skipped, not writing %s", path)
			return
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		if err := writeFixture(path, r.interactions); err != nil {
			t.Errorf("httpreplay: %v", err)
		}
	})
	return r
}

// fail reports a failed exchange on t directly, because backends sometimes
// discard request errors. That also keeps the incomplete recording from being
// written over a good fixture.
func (r *recorder) fail(err error) (*http.Response, error) {
	r.t.Errorf("%v", err)
	return nil, err
}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	reqBody, err := readRequestBody(req)
	if err != nil {
		return r.fail(err)
	}
	// RoundTrip must not modify req, so the body that was just read is sent
	// on a copy.
	out := req.Clone(req.Context())
	if reqBody != nil {
		out.Body = io.NopCloser(bytes.NewReader(reqBody))
	}

	resp, err := r.base.RoundTrip(out)
	if err != nil {
		return r.fail(fmt.Errorf("httpreplay: %s %s: %w", req.Method, req.URL, err))
	}
	respBody, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return r.fail(fmt.Errorf("httpreplay: reading response body: %w", err))
	}
	resp.Body = io.NopCloser(bytes.NewReader(respBody))

	recorded := recordedResponse{StatusCode: resp.StatusCode, Header: keptHeaders(resp.Header)}
	recorded.Body, recorded.BodyEncoding = encodeBody(respBody)

	recordedReq := recordedRequest{Method: req.Method, URL: req.URL.String()}
	recordedReq.Body, recordedReq.BodyEncoding = encodeBody(reqBody)

	r.mu.Lock()
	r.interactions = append(r.interactions, interaction{
		Request:  recordedReq,
		Response: recorded,
	})
	r.mu.Unlock()

	return resp, nil
}

// replayer serves recorded responses. Each interaction is used at most once,
// in recorded order, so repeated identical requests get their responses back
// in sequence.
type replayer struct {
	t    testing.TB
	path string

	mu           sync.Mutex
	interactions []interaction
	used         []bool
}

func newReplayer(t testing.TB, path string) (*replayer, error) {
	interactions, err := readFixture(path)
	if err != nil {
		return nil, err
	}
	r := &replayer{
		t:            t,
		path:         path,
		interactions: interactions,
		used:         make([]bool, len(interactions)),
	}
	t.Cleanup(r.checkAllUsed)
	return r, nil
}

func (r *replayer) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := readRequestBody(req)
	if err != nil {
		return nil, err
	}
	target := req.URL.String()

	r.mu.Lock()
	defer r.mu.Unlock()
	for i, in := range r.interactions {
		if r.used[i] || in.Request.Method != req.Method || in.Request.URL != target {
			continue
		}
		// An undecodable recorded body can never match a real request.
		want, err := decodeBody(in.Request.Body, in.Request.BodyEncoding)
		if err != nil || !bytes.Equal(want, body) {
			continue
		}
		resp, err := in.Response.httpResponse(req)
		if err != nil {
			r.t.Errorf("%v", err)
			return nil, err
		}
		r.used[i] = true
		return resp, nil
	}

	// Backends sometimes discard request errors, so fail the test directly
	// rather than relying on the error reaching an assertion.
	err = fmt.Errorf("httpreplay: no recorded response for %s %s in %s", req.Method, target, r.path)
	r.t.Errorf("%v", err)
	return nil, err
}

func (r *replayer) checkAllUsed() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, in := range r.interactions {
		if !r.used[i] {
			r.t.Errorf("httpreplay: recorded request %s %s in %s was never made", in.Request.Method, in.Request.URL, r.path)
		}
	}
}

// keptHeaders returns the response headers worth storing. Everything else
// changes on every request or identifies where the recording was made, which
// would rewrite every fixture on each re-record.
func keptHeaders(h http.Header) http.Header {
	kept := http.Header{}
	for name, values := range h {
		if keepHeader(name) {
			kept[name] = slices.Clone(values)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}

// keepHeader reports whether the SDKs read the response header name. Add a
// header here when a test needs one that is not listed.
func keepHeader(name string) bool {
	name = http.CanonicalHeaderKey(name)
	switch name {
	case "Content-Type", "Link", "Location", "Retry-After",
		"X-Total", "X-Total-Count", "X-Total-Pages",
		"X-Page", "X-Per-Page", "X-Next-Page", "X-Prev-Page":
		return true
	}
	// X-Gitlab-Meta carries a correlation ID that differs on every request.
	return strings.HasPrefix(name, "X-Gitlab-") && name != "X-Gitlab-Meta"
}

func (resp recordedResponse) httpResponse(req *http.Request) (*http.Response, error) {
	body, err := decodeBody(resp.Body, resp.BodyEncoding)
	if err != nil {
		return nil, err
	}
	header := resp.Header.Clone()
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{
		Status:        strconv.Itoa(resp.StatusCode) + " " + http.StatusText(resp.StatusCode),
		StatusCode:    resp.StatusCode,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}, nil
}

func readRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}
	body, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("httpreplay: reading request body: %w", err)
	}
	return body, nil
}

func encodeBody(body []byte) (text, encoding string) {
	if utf8.Valid(body) {
		return string(body), ""
	}
	return base64.StdEncoding.EncodeToString(body), "base64"
}

func decodeBody(text, encoding string) ([]byte, error) {
	switch encoding {
	case "":
		return []byte(text), nil
	case "base64":
		return base64.StdEncoding.DecodeString(text)
	default:
		return nil, fmt.Errorf("httpreplay: unknown body encoding %q", encoding)
	}
}

func readFixture(path string) ([]interaction, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no recording at %s", path)
	}
	if err != nil {
		return nil, err
	}
	var interactions []interaction
	if err := json.Unmarshal(data, &interactions); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return interactions, nil
}

func writeFixture(path string, interactions []interaction) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(interactions); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

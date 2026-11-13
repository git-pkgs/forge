package httpreplay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// fakeTB collects errors and cleanups so tests can check how the transports
// report problems without failing the real test.
type fakeTB struct {
	testing.TB
	errors   []string
	cleanups []func()
}

func (f *fakeTB) Helper()             {}
func (f *fakeTB) Logf(string, ...any) {}
func (f *fakeTB) Failed() bool        { return len(f.errors) > 0 }
func (f *fakeTB) Cleanup(fn func())   { f.cleanups = append(f.cleanups, fn) }

func (f *fakeTB) Errorf(format string, args ...any) {
	f.errors = append(f.errors, fmt.Sprintf(format, args...))
}

func (f *fakeTB) finish() {
	for i := len(f.cleanups) - 1; i >= 0; i-- {
		f.cleanups[i]()
	}
}

const testToken = "secret-token-value"

func newCountingServer(t *testing.T) *httptest.Server {
	t.Helper()
	var hits atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/octocat/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", `<https://api.example.test/repos?page=2>; rel="next"`)
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "cookie-value"})
		_, _ = w.Write([]byte(`{"name":"hello","html":"<b>&</b>"}`))
	})
	mux.HandleFunc("POST /repos/octocat/hello/issues", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(append([]byte("created:"), body...))
	})
	mux.HandleFunc("GET /counter", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strconv.FormatInt(hits.Add(1), 10)))
	})
	mux.HandleFunc("GET /binary", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte{0xff, 0x00, 0xfe})
	})
	mux.HandleFunc("GET /missing", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

type exchange struct {
	method, path, body string
	wantStatus         int
	wantBody           []byte
}

func exchanges() []exchange {
	return []exchange{
		{method: "GET", path: "/repos/octocat/hello", wantStatus: 200, wantBody: []byte(`{"name":"hello","html":"<b>&</b>"}`)},
		{method: "POST", path: "/repos/octocat/hello/issues", body: `{"title":"a"}`, wantStatus: 201, wantBody: []byte(`created:{"title":"a"}`)},
		{method: "POST", path: "/repos/octocat/hello/issues", body: `{"title":"b"}`, wantStatus: 201, wantBody: []byte(`created:{"title":"b"}`)},
		{method: "GET", path: "/counter", wantStatus: 200, wantBody: []byte("1")},
		{method: "GET", path: "/counter", wantStatus: 200, wantBody: []byte("2")},
		{method: "GET", path: "/binary", wantStatus: 200, wantBody: []byte{0xff, 0x00, 0xfe}},
		{method: "GET", path: "/missing", wantStatus: 404, wantBody: []byte("404 page not found\n")},
	}
}

func do(t *testing.T, hc *http.Client, baseURL string, ex exchange) *http.Response {
	t.Helper()
	var body io.Reader
	if ex.body != "" {
		body = strings.NewReader(ex.body)
	}
	req, err := http.NewRequest(ex.method, baseURL+ex.path, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+testToken)
	resp, err := hc.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", ex.method, ex.path, err)
	}
	got, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != ex.wantStatus {
		t.Errorf("%s %s: status = %d, want %d", ex.method, ex.path, resp.StatusCode, ex.wantStatus)
	}
	if !bytes.Equal(got, ex.wantBody) {
		t.Errorf("%s %s: body = %q, want %q", ex.method, ex.path, got, ex.wantBody)
	}
	return resp
}

func TestRecordThenReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testdata", "replay", "TestExample.json")
	srv := newCountingServer(t)

	t.Run("record", func(t *testing.T) {
		hc := &http.Client{Transport: newRecorder(t, path, http.DefaultTransport)}
		for _, ex := range exchanges() {
			do(t, hc, srv.URL, ex)
		}
	})

	fixture, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("fixture not written: %v", err)
	}
	for _, leaked := range []string{testToken, "cookie-value", "Set-Cookie"} {
		if bytes.Contains(fixture, []byte(leaked)) {
			t.Errorf("fixture contains %q:\n%s", leaked, fixture)
		}
	}
	if !bytes.Contains(fixture, []byte(`<b>&</b>`)) {
		t.Errorf("fixture escapes HTML in bodies:\n%s", fixture)
	}

	// Replay with the server gone to prove nothing reaches the network.
	srv.Close()

	t.Run("replay", func(t *testing.T) {
		r, err := newReplayer(t, path)
		if err != nil {
			t.Fatal(err)
		}
		hc := &http.Client{Transport: r}
		for _, ex := range exchanges() {
			resp := do(t, hc, srv.URL, ex)
			if ex.path == "/repos/octocat/hello" {
				if got := resp.Header.Get("Link"); !strings.Contains(got, `rel="next"`) {
					t.Errorf("Link header = %q, want the recorded value", got)
				}
				if got := resp.Header.Get("Content-Type"); got != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", got)
				}
			}
		}
	})
}

func writeTestFixture(t *testing.T, interactions []interaction) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.json")
	if err := writeFixture(path, interactions); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReplayUnmatchedRequestFailsTest(t *testing.T) {
	path := writeTestFixture(t, []interaction{{
		Request:  recordedRequest{Method: "GET", URL: "https://api.example.test/a"},
		Response: recordedResponse{StatusCode: 200, Body: "a"},
	}})
	ft := &fakeTB{}
	r, err := newReplayer(ft, path)
	if err != nil {
		t.Fatal(err)
	}
	hc := &http.Client{Transport: r}

	for _, target := range []string{"https://api.example.test/a", "https://api.example.test/a", "https://api.example.test/b"} {
		resp, err := hc.Get(target)
		if err == nil {
			_ = resp.Body.Close()
		}
	}
	ft.finish()

	want := []string{
		"httpreplay: no recorded response for GET https://api.example.test/a in " + path,
		"httpreplay: no recorded response for GET https://api.example.test/b in " + path,
	}
	if len(ft.errors) != len(want) {
		t.Fatalf("errors = %q, want %q", ft.errors, want)
	}
	for i := range want {
		if ft.errors[i] != want[i] {
			t.Errorf("errors[%d] = %q, want %q", i, ft.errors[i], want[i])
		}
	}
}

func TestReplayMatchesRequestBody(t *testing.T) {
	path := writeTestFixture(t, []interaction{{
		Request:  recordedRequest{Method: "POST", URL: "https://api.example.test/issues", Body: `{"title":"a"}`},
		Response: recordedResponse{StatusCode: 201},
	}})
	ft := &fakeTB{}
	r, err := newReplayer(ft, path)
	if err != nil {
		t.Fatal(err)
	}
	hc := &http.Client{Transport: r}

	resp, err := hc.Post("https://api.example.test/issues", "application/json", strings.NewReader(`{"title":"b"}`))
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("request with a different body matched the recording")
	}
	if len(ft.errors) != 1 || !strings.Contains(ft.errors[0], "no recorded response for POST") {
		t.Errorf("errors = %q, want one unmatched POST", ft.errors)
	}
}

func TestReplayUnusedInteractionFailsTest(t *testing.T) {
	path := writeTestFixture(t, []interaction{
		{Request: recordedRequest{Method: "GET", URL: "https://api.example.test/a"}, Response: recordedResponse{StatusCode: 200}},
		{Request: recordedRequest{Method: "GET", URL: "https://api.example.test/b"}, Response: recordedResponse{StatusCode: 200}},
	})
	ft := &fakeTB{}
	r, err := newReplayer(ft, path)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := (&http.Client{Transport: r}).Get("https://api.example.test/a")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	ft.finish()

	want := "httpreplay: recorded request GET https://api.example.test/b in " + path + " was never made"
	if len(ft.errors) != 1 || ft.errors[0] != want {
		t.Errorf("errors = %q, want [%q]", ft.errors, want)
	}
}

func TestRecorderSkipsWriteWhenTestFails(t *testing.T) {
	srv := newCountingServer(t)
	path := filepath.Join(t.TempDir(), "fixture.json")
	ft := &fakeTB{}
	hc := &http.Client{Transport: newRecorder(ft, path, http.DefaultTransport)}
	resp, err := hc.Get(srv.URL + "/counter")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	ft.errors = append(ft.errors, "assertion failed")
	ft.finish()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("fixture written for a failed test: stat err = %v", err)
	}
}

func TestRecorderWritesEmptyFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.json")
	ft := &fakeTB{}
	newRecorder(ft, path, http.DefaultTransport)
	ft.finish()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "[]\n" {
		t.Errorf("fixture = %q, want %q", got, "[]\n")
	}
}

func TestNewReplayerErrors(t *testing.T) {
	dir := t.TempDir()

	missing := filepath.Join(dir, "missing.json")
	if _, err := newReplayer(&fakeTB{}, missing); err == nil || err.Error() != "no recording at "+missing {
		t.Errorf("missing fixture: err = %v", err)
	}

	invalid := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(invalid, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := newReplayer(&fakeTB{}, invalid); err == nil || !strings.HasPrefix(err.Error(), "parsing "+invalid) {
		t.Errorf("invalid fixture: err = %v", err)
	}
}

func TestReplayUnknownBodyEncoding(t *testing.T) {
	path := writeTestFixture(t, []interaction{{
		Request:  recordedRequest{Method: "GET", URL: "https://api.example.test/a"},
		Response: recordedResponse{StatusCode: 200, Body: "a", BodyEncoding: "gzip"},
	}})
	r, err := newReplayer(&fakeTB{}, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&http.Client{Transport: r}).Get("https://api.example.test/a"); err == nil || !strings.Contains(err.Error(), `unknown body encoding "gzip"`) {
		t.Errorf("err = %v, want unknown body encoding", err)
	}
}

func TestFixturePath(t *testing.T) {
	t.Run("sub test", func(t *testing.T) {
		want := filepath.Join("testdata", "replay", "TestFixturePath", "sub_test.json")
		if got := fixturePath(t); got != want {
			t.Errorf("fixturePath = %q, want %q", got, want)
		}
	})
}

func TestClientRecordsWhenEnabled(t *testing.T) {
	srv := newCountingServer(t)
	dir := t.TempDir()

	t.Run("record", func(t *testing.T) {
		t.Setenv(recordEnv, "1")
		t.Chdir(dir)
		hc := Client(t)
		do(t, hc, srv.URL, exchange{method: "GET", path: "/counter", wantStatus: 200, wantBody: []byte("1")})
	})

	path := filepath.Join(dir, "testdata", "replay", "TestClientRecordsWhenEnabled", "record.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture not written: %v", err)
	}
}

// TestClientReplaysFixture replays testdata/replay/TestClientReplaysFixture.json.
func TestClientReplaysFixture(t *testing.T) {
	t.Setenv(recordEnv, "")
	hc := Client(t)
	do(t, hc, "https://api.example.test", exchange{method: "GET", path: "/hello", wantStatus: 200, wantBody: []byte("hello from the fixture")})
}

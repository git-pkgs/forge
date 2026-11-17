package forges

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// recordAuth answers every request with 204 and records the Authorization
// header sent to each URL.
type recordAuth struct {
	auth map[string][]string
}

func (rt *recordAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.auth == nil {
		rt.auth = make(map[string][]string)
	}
	rt.auth[req.URL.String()] = req.Header.Values("Authorization")
	return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: http.NoBody, Request: req}, nil
}

func TestAuthHTTPClientAuthenticatesEachRequest(t *testing.T) {
	var calls atomic.Int32
	creds := CredentialsFunc(func(req *http.Request) error {
		n := calls.Add(1)
		req.Header.Set("Authorization", "Bearer token-"+strconv.Itoa(int(n)))
		return nil
	})
	var mu sync.Mutex
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, r.Header.Get("Authorization"))
	}))
	defer srv.Close()

	hc := AuthHTTPClient(srv.Client(), creds)
	for range 2 {
		req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := hc.Do(req)
		if err != nil {
			t.Fatalf("Do: %v", err)
		}
		_ = resp.Body.Close()
		if h := req.Header.Get("Authorization"); h != "" {
			t.Errorf("caller's request was modified: Authorization = %q", h)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if want := []string{"Bearer token-1", "Bearer token-2"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("Authorization headers = %q, want %q", got, want)
	}
}

func TestAuthHTTPClientAuthenticatesRedirects(t *testing.T) {
	var mu sync.Mutex
	var got []string
	record := func(r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, r.Header.Get("Authorization"))
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/old", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		http.Redirect(w, r, "/new", http.StatusFound)
	})
	mux.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		record(r)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	hc := AuthHTTPClient(srv.Client(), BearerToken(srv.Listener.Addr().String(), TokenSourceFunc(func(context.Context) (string, error) {
		return "secret", nil
	})))
	resp, err := hc.Get(srv.URL + "/old")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	_ = resp.Body.Close()
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || got[0] != "Bearer secret" || got[1] != "Bearer secret" {
		t.Errorf("Authorization headers = %q, want the token on both requests", got)
	}
}

func TestAuthHTTPClientRefusesOnError(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
	}))
	defer srv.Close()

	errNoToken := errors.New("no token")
	hc := AuthHTTPClient(srv.Client(), CredentialsFunc(func(*http.Request) error { return errNoToken }))
	body := &closeRecorder{Reader: strings.NewReader(`{"title":"x"}`)}
	req, err := http.NewRequest(http.MethodPost, srv.URL, body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := hc.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected an error")
	}
	if !errors.Is(err, errNoToken) {
		t.Errorf("err = %v, want it to wrap the credentials error", err)
	}
	if !body.closed {
		t.Error("request body was not closed")
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("server received %d requests, want 0", n)
	}
}

func TestAuthHTTPClientCopiesClient(t *testing.T) {
	base := &recordAuth{}
	hc := &http.Client{Transport: base, Timeout: 5 * time.Second}
	creds := CredentialsFunc(func(*http.Request) error { return nil })

	got := AuthHTTPClient(hc, creds)
	if got == hc {
		t.Fatal("AuthHTTPClient returned the caller's client")
	}
	if hc.Transport != base {
		t.Error("AuthHTTPClient changed the caller's transport")
	}
	if got.Timeout != hc.Timeout {
		t.Errorf("Timeout = %v, want %v", got.Timeout, hc.Timeout)
	}

	if AuthHTTPClient(nil, creds).Transport.(*authTransport).base != http.DefaultTransport {
		t.Error("nil client should wrap http.DefaultTransport")
	}
}

func TestBearerTokenScopesToHost(t *testing.T) {
	rt := &recordAuth{}
	hc := AuthHTTPClient(&http.Client{Transport: rt}, BearerToken("GitHub.com", TokenSourceFunc(func(context.Context) (string, error) {
		return "secret", nil
	})))

	want := map[string]string{
		"https://github.com/user":                         "Bearer secret",
		"https://api.github.com/user":                     "Bearer secret",
		"https://API.GitHub.com:443/user":                 "Bearer secret",
		"https://uploads.github.com/repos/o/r/releases":   "Bearer secret",
		"https://evilgithub.com/user":                     "",
		"https://github.com.example.com/user":             "",
		"https://tmaproduction.blob.core.windows.net/b/1": "",
		"https://gitlab.com/api/v4/user":                  "",
	}
	for u := range want {
		resp, err := hc.Get(u)
		if err != nil {
			t.Fatalf("Get(%q): %v", u, err)
		}
		_ = resp.Body.Close()
	}
	for u, auth := range want {
		got := strings.Join(rt.auth[u], ",")
		if got != auth {
			t.Errorf("%s: Authorization = %q, want %q", u, got, auth)
		}
	}
}

func TestBearerTokenHostForms(t *testing.T) {
	const u = "http://git.example.com:3000/api/v1/user"
	for _, host := range []string{"git.example.com", "git.example.com:3000", "http://git.example.com:3000", "https://git.example.com/", "Git.Example.COM"} {
		rt := &recordAuth{}
		hc := AuthHTTPClient(&http.Client{Transport: rt}, BearerToken(host, TokenSourceFunc(func(context.Context) (string, error) {
			return "secret", nil
		})))
		resp, err := hc.Get(u)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if got := strings.Join(rt.auth[u], ","); got != "Bearer secret" {
			t.Errorf("BearerToken(%q): Authorization = %q, want the token", host, got)
		}
	}
	for _, host := range []string{"", "://"} {
		rt := &recordAuth{}
		hc := AuthHTTPClient(&http.Client{Transport: rt}, BearerToken(host, TokenSourceFunc(func(context.Context) (string, error) {
			return "secret", nil
		})))
		resp, err := hc.Get(u)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if got := rt.auth[u]; len(got) != 0 {
			t.Errorf("BearerToken(%q): Authorization = %q, want no header", host, got)
		}
	}
}

func TestBearerTokenEmptyTokenSendsNoHeader(t *testing.T) {
	rt := &recordAuth{}
	hc := AuthHTTPClient(&http.Client{Transport: rt}, BearerToken("github.com", TokenSourceFunc(func(context.Context) (string, error) {
		return "", nil
	})))
	resp, err := hc.Get("https://api.github.com/repos/o/r")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if got := rt.auth["https://api.github.com/repos/o/r"]; len(got) != 0 {
		t.Errorf("Authorization = %q, want no header", got)
	}
}

func TestBearerTokenSourceError(t *testing.T) {
	errExpired := errors.New("refresh failed")
	rt := &recordAuth{}
	var gotCtx atomic.Value
	type ctxKey struct{}
	hc := AuthHTTPClient(&http.Client{Transport: rt}, BearerToken("github.com", TokenSourceFunc(func(ctx context.Context) (string, error) {
		gotCtx.Store(ctx.Value(ctxKey{}))
		return "", errExpired
	})))

	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := hc.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected an error")
	}
	if !errors.Is(err, errExpired) {
		t.Errorf("err = %v, want it to wrap the token source error", err)
	}
	if !strings.Contains(err.Error(), "token for github.com") {
		t.Errorf("err = %v, want it to name the host", err)
	}
	if gotCtx.Load() != "request" {
		t.Error("token source did not receive the request's context")
	}
	if len(rt.auth) != 0 {
		t.Errorf("requests sent: %v, want none", rt.auth)
	}
}

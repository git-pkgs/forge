package factory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/git-pkgs/forge"
)

type stubForge struct {
	forges.Forge
}

// recordingTransport answers every request with a 404 and records the
// Authorization header sent to each host.
type recordingTransport struct {
	mu   sync.Mutex
	auth map[string]string
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.auth == nil {
		rt.auth = make(map[string]string)
	}
	rt.auth[req.URL.Host] = req.Header.Get("Authorization")
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     make(http.Header),
		Body:       http.NoBody,
		Request:    req,
	}, nil
}

func TestNewClientRegistersDefaultForges(t *testing.T) {
	c := NewClient()
	for domain, wantAPI := range map[string]string{
		"github.com":    "https://api.github.com",
		"gitlab.com":    "https://gitlab.com/api/v4",
		"codeberg.org":  "https://codeberg.org/api/v1",
		"bitbucket.org": "https://api.bitbucket.org/2.0",
		"tangled.org":   "https://tangled.org/xrpc",
	} {
		f, err := c.ForgeFor(domain)
		if err != nil {
			t.Errorf("ForgeFor(%q): %v", domain, err)
			continue
		}
		provider, ok := f.(forges.APIBaseURLProvider)
		if !ok {
			t.Errorf("forge for %q does not expose its API URL", domain)
			continue
		}
		if got := provider.APIBaseURL(); got != wantAPI {
			t.Errorf("API URL for %q = %q, want %q", domain, got, wantAPI)
		}
	}
}

func TestNewClientAppliesOptionsToDefaultForges(t *testing.T) {
	rt := &recordingTransport{}
	c := NewClient(
		forges.WithHTTPClient(&http.Client{Transport: rt}),
		forges.WithToken("codeberg.org", "secret"),
	)

	f, owner, repo, err := c.ForgeForURL(context.Background(), "https://codeberg.org/owner/repo")
	if err != nil {
		t.Fatalf("ForgeForURL: %v", err)
	}
	if owner != "owner" || repo != "repo" {
		t.Errorf("owner/repo = %q/%q", owner, repo)
	}
	if _, err := f.Repos().Get(context.Background(), owner, repo); err == nil {
		t.Fatal("expected not found error from recording transport")
	}
	if got := rt.auth["codeberg.org"]; got != "token secret" {
		t.Errorf("Authorization = %q, want the configured token", got)
	}
}

func TestNewClientKeepsRegisteredForge(t *testing.T) {
	want := &stubForge{}
	c := NewClient(forges.WithForge("github.com", want))
	got, err := c.ForgeFor("github.com")
	if err != nil {
		t.Fatalf("ForgeFor: %v", err)
	}
	if got != forges.Forge(want) {
		t.Errorf("ForgeFor returned %T, want the registered forge", got)
	}
}

func TestDefaultForgesAskForEachToken(t *testing.T) {
	var asked []string
	defaults := DefaultForges(func(domain string) string {
		asked = append(asked, domain)
		return ""
	}, nil)
	if len(asked) != len(defaults) {
		t.Errorf("asked for tokens for %v, want one per default forge (%d)", asked, len(defaults))
	}
	for _, domain := range asked {
		if defaults[domain] == nil {
			t.Errorf("no default forge for %q", domain)
		}
	}
}

func TestForgeForURLDetectsUnknownDomainOnce(t *testing.T) {
	var probes, apiCalls atomic.Int32
	var gotAuth atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			probes.Add(1)
			w.Header().Set("X-Forgejo-Version", "7.0.0")
			return
		}
		apiCalls.Add(1)
		gotAuth.Store(r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"full_name": "owner/repo",
			"name":      "repo",
			"owner":     map[string]any{"login": "owner"},
		})
	}))
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "http://")

	c := NewClient(forges.WithToken(host, "secret"))
	ctx := context.Background()
	repoURL := srv.URL + "/owner/repo"

	if _, _, _, err := c.ForgeForURL(ctx, repoURL); err != nil {
		t.Fatalf("ForgeForURL: %v", err)
	}
	r, err := c.FetchRepository(ctx, repoURL)
	if err != nil {
		t.Fatalf("FetchRepository: %v", err)
	}
	if r.FullName != "owner/repo" {
		t.Errorf("FullName = %q", r.FullName)
	}
	if probes.Load() != 1 {
		t.Errorf("detection probes = %d, want 1", probes.Load())
	}
	if apiCalls.Load() == 0 {
		t.Error("expected the detected forge to reach the API")
	}
	if got := gotAuth.Load(); got != "token secret" {
		t.Errorf("Authorization = %q, want the configured token", got)
	}
	if _, err := c.ForgeFor(host); err != nil {
		t.Errorf("ForgeFor(%q) after detection: %v", host, err)
	}
}

func TestForgeForURLReportsUndetectedDomain(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	c := NewClient()
	_, _, _, err := c.ForgeForURL(context.Background(), srv.URL+"/owner/repo")
	if err == nil || !strings.Contains(err.Error(), "could not detect forge type") {
		t.Fatalf("ForgeForURL error = %v, want a detection failure", err)
	}
}

func TestForgeForURLDetectsConcurrently(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Gitea-Version", "1.22.0")
	}))
	defer srv.Close()

	c := NewClient()
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if _, _, _, err := c.ForgeForURL(context.Background(), srv.URL+"/owner/repo"); err != nil {
				t.Errorf("ForgeForURL: %v", err)
			}
			_ = c.Tokens()
		})
	}
	wg.Wait()
}

package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	forge "github.com/git-pkgs/forge"
	"github.com/golang/snappy"
	"github.com/google/go-github/v82/github"
)

const (
	testAttestationDigest = "sha256:5db1fee4b5703808c48078a76768b155b421b210c0761cd6a5d223f4d99f1eaa"
	testAttestationPath   = "/api/v3/repos/octocat/hello/attestations/" + testAttestationDigest
	testBundle            = `{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json","dsseEnvelope":{"payloadType":"application/vnd.in-toto+json"}}`
)

func newTestGitHubAttestationService(srv *httptest.Server) *gitHubAttestationService {
	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	return &gitHubAttestationService{client: c}
}

func TestGitHubListAttestationsPaginates(t *testing.T) {
	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+testAttestationPath, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assertEqual(t, "per_page", "100", q.Get("per_page"))
		w.Header().Set("Content-Type", "application/json")
		switch q.Get("after") {
		case "":
			w.Header().Set("Link", fmt.Sprintf(`<%s%s?per_page=100&after=cursor1>; rel="next"`, srvURL, testAttestationPath))
			_, _ = fmt.Fprintf(w, `{"attestations": [
				{"repository_id": 1, "bundle": %s, "bundle_url": "https://blob.example.com/1.json.sn"},
				{"repository_id": 1, "bundle": null, "bundle_url": "https://blob.example.com/2.json.sn"}
			]}`, testBundle)
		case "cursor1":
			_, _ = fmt.Fprint(w, `{"attestations": [{"repository_id": 1, "bundle_url": "https://blob.example.com/3.json.sn"}]}`)
		default:
			t.Errorf("unexpected after cursor %q", q.Get("after"))
		}
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL

	svc := newTestGitHubAttestationService(srv)
	atts, err := svc.List(context.Background(), "octocat", "hello", testAttestationDigest[:7]+strings.ToUpper(testAttestationDigest[7:]), forge.ListAttestationOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(atts) != 3 {
		t.Fatalf("expected 3 attestations, got %d", len(atts))
	}
	for i, att := range atts {
		assertEqual(t, fmt.Sprintf("SubjectDigest[%d]", i), testAttestationDigest, att.SubjectDigest)
		assertEqual(t, fmt.Sprintf("BundleURL[%d]", i), fmt.Sprintf("https://blob.example.com/%d.json.sn", i+1), att.BundleURL)
		assertEqual(t, fmt.Sprintf("ID[%d]", i), "", att.ID)
	}
	assertEqual(t, "Bundle[0]", testBundle, string(atts[0].Bundle))
	if atts[1].Bundle != nil || atts[2].Bundle != nil {
		t.Errorf("expected no inline bundle for null or missing bundles, got %q and %q", atts[1].Bundle, atts[2].Bundle)
	}
}

func TestGitHubListAttestationsPredicateTypeAndLimit(t *testing.T) {
	var srvURL string
	var requests atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+testAttestationPath, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assertEqual(t, "predicate_type", "https://slsa.dev/provenance/v1", r.URL.Query().Get("predicate_type"))
		assertEqual(t, "per_page", "2", r.URL.Query().Get("per_page"))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", fmt.Sprintf(`<%s%s?per_page=2&after=cursor1>; rel="next"`, srvURL, testAttestationPath))
		_, _ = fmt.Fprint(w, `{"attestations": [{"bundle_url": "https://blob.example.com/1.json.sn"}, {"bundle_url": "https://blob.example.com/2.json.sn"}]}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL

	svc := newTestGitHubAttestationService(srv)
	atts, err := svc.List(context.Background(), "octocat", "hello", testAttestationDigest, forge.ListAttestationOpts{
		PredicateType: "https://slsa.dev/provenance/v1",
		Limit:         2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(atts) != 2 {
		t.Fatalf("expected 2 attestations, got %d", len(atts))
	}
	if n := requests.Load(); n != 1 {
		t.Errorf("expected 1 request once the limit was reached, got %d", n)
	}
}

func TestGitHubListAttestationsEmpty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+testAttestationPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"attestations": []}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubAttestationService(srv)
	atts, err := svc.List(context.Background(), "octocat", "hello", testAttestationDigest, forge.ListAttestationOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(atts) != 0 {
		t.Errorf("expected no attestations, got %d", len(atts))
	}
}

func TestGitHubListAttestationsErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		headers map[string]string
		want    error
	}{
		{name: "not found", status: http.StatusNotFound, want: forge.ErrNotFound},
		{name: "unauthorized", status: http.StatusUnauthorized, want: forge.ErrUnauthorized},
		{
			name:    "rate limited",
			status:  http.StatusForbidden,
			headers: map[string]string{"X-RateLimit-Limit": "60", "X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1700000000"},
			want:    forge.ErrRateLimited,
		},
		{name: "server error", status: http.StatusBadGateway, want: forge.ErrServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("GET "+testAttestationPath, func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.headers {
					w.Header().Set(k, v)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprint(w, `{"message": "error"}`)
			})

			srv := httptest.NewServer(mux)
			defer srv.Close()

			svc := newTestGitHubAttestationService(srv)
			atts, err := svc.List(context.Background(), "octocat", "hello", testAttestationDigest, forge.ListAttestationOpts{})
			if err == nil {
				t.Fatalf("expected an error, got %d attestations", len(atts))
			}
			if !errors.Is(err, tt.want) {
				t.Errorf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestGitHubListAttestationsInvalidDigest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s", r.URL.Path)
	}))
	defer srv.Close()

	svc := newTestGitHubAttestationService(srv)
	_, err := svc.List(context.Background(), "octocat", "hello", "sha256:../../user", forge.ListAttestationOpts{})
	if !errors.Is(err, forge.ErrInvalidDigest) {
		t.Errorf("expected ErrInvalidDigest, got %v", err)
	}
}

func TestGitHubDownloadAttestationFromBundleURL(t *testing.T) {
	blob := http.NewServeMux()
	blob.HandleFunc("GET /attestations/1.json.sn", func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("bundle request sent Authorization %q", auth)
		}
		w.Header().Set("Content-Type", "application/x-snappy")
		_, _ = w.Write(snappy.Encode(nil, []byte(testBundle)))
	})
	blobSrv := httptest.NewTLSServer(blob)
	defer blobSrv.Close()

	f := NewWithBase("https://ghe.example.com", "secret-token", blobSrv.Client())
	bundle, err := f.Attestations().Download(context.Background(), "octocat", "hello", forge.Attestation{
		SubjectDigest: testAttestationDigest,
		BundleURL:     blobSrv.URL + "/attestations/1.json.sn?sig=signed",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "bundle", testBundle, string(bundle))
}

func TestGitHubDownloadAttestationInlineBundle(t *testing.T) {
	svc := &gitHubAttestationService{client: github.NewClient(nil)}
	bundle, err := svc.Download(context.Background(), "octocat", "hello", forge.Attestation{
		Bundle:    []byte(testBundle),
		BundleURL: "http://127.0.0.1:0/unreachable",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "bundle", testBundle, string(bundle))
}

func TestGitHubDownloadAttestationErrors(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		limit   int
		want    error
	}{
		{
			name: "unauthorized",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			want: forge.ErrUnauthorized,
		},
		{
			name: "rate limited",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
			},
			want: forge.ErrRateLimited,
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			want: forge.ErrServerError,
		},
		{
			name: "not snappy",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
			},
		},
		{
			name: "not json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(snappy.Encode(nil, []byte("not json")))
			},
		},
		{
			name: "decompressed too large",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(snappy.Encode(nil, []byte(strings.Repeat(" ", 1024)+testBundle)))
			},
			limit: 512,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.limit > 0 {
				old := maxBundleBytes
				maxBundleBytes = tt.limit
				t.Cleanup(func() { maxBundleBytes = old })
			}
			srv := httptest.NewTLSServer(tt.handler)
			defer srv.Close()

			svc := &gitHubAttestationService{client: github.NewClient(nil), httpClient: srv.Client()}
			_, err := svc.Download(context.Background(), "octocat", "hello", forge.Attestation{
				BundleURL: srv.URL + "/1.json.sn?sig=signed-secret",
			})
			if err == nil {
				t.Fatal("expected an error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Errorf("expected %v, got %v", tt.want, err)
			}
			if strings.Contains(err.Error(), "signed-secret") {
				t.Errorf("error leaks the signed bundle URL: %v", err)
			}
		})
	}
}

func TestGitHubDownloadAttestationNoBundle(t *testing.T) {
	svc := &gitHubAttestationService{client: github.NewClient(nil)}
	if _, err := svc.Download(context.Background(), "octocat", "hello", forge.Attestation{}); err == nil {
		t.Error("expected an error for an attestation without a bundle or bundle URL")
	}
}

func TestGitHubDownloadAttestationRejectsHTTPBundleURL(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write(snappy.Encode(nil, []byte(testBundle)))
	}))
	defer srv.Close()

	// The forge's own API is https, so a plain http bundle URL is refused.
	f := NewWithBase("https://ghe.example.com", "secret-token", srv.Client())
	_, err := f.Attestations().Download(context.Background(), "octocat", "hello", forge.Attestation{
		BundleURL: srv.URL + "/1.json.sn?sig=signed-secret",
	})
	if err == nil {
		t.Fatal("expected an error for an http bundle URL")
	}
	if strings.Contains(err.Error(), "signed-secret") {
		t.Errorf("error leaks the signed bundle URL: %v", err)
	}
	if hits.Load() != 0 {
		t.Errorf("bundle server was contacted %d times", hits.Load())
	}
}

func TestGitHubDownloadAttestationAllowsHTTPBundleURLOnHTTPForge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(snappy.Encode(nil, []byte(testBundle)))
	}))
	defer srv.Close()

	f := NewWithBase(srv.URL, "secret-token", nil)
	bundle, err := f.Attestations().Download(context.Background(), "octocat", "hello", forge.Attestation{
		BundleURL: srv.URL + "/1.json.sn?sig=signed",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "bundle", testBundle, string(bundle))
}

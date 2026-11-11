//go:build !tinygo

package gitlab

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	forge "github.com/git-pkgs/forge"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const (
	testAttestationHex    = "5db1fee4b5703808c48078a76768b155b421b210c0761cd6a5d223f4d99f1eaa"
	testAttestationDigest = "sha256:" + testAttestationHex
	testAttestationPath   = "/api/v4/projects/group%2Fproject/attestations/" + testAttestationHex
	testBundle            = `{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json","dsseEnvelope":{"payloadType":"application/vnd.in-toto+json"}}`
)

func gitLabAttestationJSON(iid int, kind, predicateType string) string {
	return fmt.Sprintf(`{
		"id": %d, "iid": %d, "project_id": 1, "build_id": 9, "status": "success",
		"created_at": "2025-10-07T20:59:27.085Z", "updated_at": "2025-10-07T20:59:27.085Z",
		"expire_at": "2027-10-07T20:59:26.967Z",
		"predicate_kind": %q, "predicate_type": %q, "subject_digest": %q,
		"download_url": "https://gitlab.example.com/api/v4/projects/1/attestations/%d/download"
	}`, iid+100, iid, kind, predicateType, testAttestationHex, iid)
}

func newTestGitLabAttestationService(t *testing.T, srv *httptest.Server) forge.AttestationService {
	t.Helper()
	c, err := gitlab.NewClient("test-token", gitlab.WithBaseURL(srv.URL+"/api/v4"), gitlab.WithoutRetries())
	if err != nil {
		t.Fatalf("creating client: %v", err)
	}
	return &gitLabAttestationService{client: c}
}

func TestGitLabListAttestationsPaginates(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+testAttestationPath, func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "PRIVATE-TOKEN", "test-token", r.Header.Get("PRIVATE-TOKEN"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page") {
		case "":
			w.Header().Set("X-Next-Page", "2")
			_, _ = fmt.Fprintf(w, "[%s, %s]",
				gitLabAttestationJSON(1, "provenance", "https://slsa.dev/provenance/v1"),
				gitLabAttestationJSON(2, "sbom", "https://cyclonedx.org/bom"))
		case "2":
			_, _ = fmt.Fprintf(w, "[%s]", gitLabAttestationJSON(3, "provenance", "https://slsa.dev/provenance/v1"))
		default:
			t.Errorf("unexpected page %q", r.URL.Query().Get("page"))
		}
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	upper := "sha256:" + strings.ToUpper(testAttestationHex)
	atts, err := f.Attestations().List(context.Background(), "group", "project", upper, forge.ListAttestationOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(atts) != 3 {
		t.Fatalf("expected 3 attestations, got %d", len(atts))
	}

	assertEqual(t, "ID[0]", "1", atts[0].ID)
	assertEqual(t, "SubjectDigest[0]", testAttestationDigest, atts[0].SubjectDigest)
	assertEqual(t, "PredicateType[0]", "https://slsa.dev/provenance/v1", atts[0].PredicateType)
	assertEqual(t, "Status[0]", "success", atts[0].Status)
	assertEqual(t, "BundleURL[0]", "https://gitlab.example.com/api/v4/projects/1/attestations/1/download", atts[0].BundleURL)
	if want := time.Date(2025, 10, 7, 20, 59, 27, 85000000, time.UTC); !atts[0].CreatedAt.Equal(want) {
		t.Errorf("CreatedAt[0]: want %v, got %v", want, atts[0].CreatedAt)
	}
	if want := time.Date(2027, 10, 7, 20, 59, 26, 967000000, time.UTC); !atts[0].ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt[0]: want %v, got %v", want, atts[0].ExpiresAt)
	}
	if atts[0].Bundle != nil {
		t.Errorf("Bundle[0]: want nil, got %s", atts[0].Bundle)
	}
	assertEqual(t, "ID[1]", "2", atts[1].ID)
	assertEqual(t, "PredicateType[1]", "https://cyclonedx.org/bom", atts[1].PredicateType)
	assertEqual(t, "ID[2]", "3", atts[2].ID)
}

func TestGitLabListAttestationsPredicateTypeAndLimit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+testAttestationPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, "[%s, %s, %s]",
			gitLabAttestationJSON(1, "provenance", "https://slsa.dev/provenance/v1"),
			gitLabAttestationJSON(2, "sbom", "https://cyclonedx.org/bom"),
			gitLabAttestationJSON(3, "sbom", "https://spdx.dev/Document"))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	tests := []struct {
		name string
		opts forge.ListAttestationOpts
		want []string
	}{
		{"short name", forge.ListAttestationOpts{PredicateType: "sbom"}, []string{"2", "3"}},
		{"full URI", forge.ListAttestationOpts{PredicateType: "https://slsa.dev/provenance/v1"}, []string{"1"}},
		{"no match", forge.ListAttestationOpts{PredicateType: "https://example.com/other"}, nil},
		{"limit", forge.ListAttestationOpts{Limit: 2}, []string{"1", "2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			atts, err := f.Attestations().List(context.Background(), "group", "project", testAttestationDigest, tt.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var ids []string
			for _, a := range atts {
				ids = append(ids, a.ID)
			}
			assertSliceEqual(t, "IDs", tt.want, ids)
		})
	}
}

func TestGitLabListAttestationsEmpty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+testAttestationPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `[]`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	atts, err := f.Attestations().List(context.Background(), "group", "project", testAttestationDigest, forge.ListAttestationOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(atts) != 0 {
		t.Errorf("expected no attestations, got %d", len(atts))
	}
}

func TestGitLabListAttestationsErrors(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		notFound bool
	}{
		// GitLab answers 404 when the slsa_provenance_statement flag is off.
		{"feature unavailable", http.StatusNotFound, `{"message":"404 Not Found"}`, true},
		{"unauthorized", http.StatusUnauthorized, `{"message":"401 Unauthorized"}`, false},
		{"forbidden", http.StatusForbidden, `{"message":"403 Forbidden"}`, false},
		{"rate limited", http.StatusTooManyRequests, `{"message":"Retry later"}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("GET "+testAttestationPath, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprint(w, tt.body)
			})

			srv := httptest.NewServer(mux)
			defer srv.Close()

			svc := newTestGitLabAttestationService(t, srv)
			atts, err := svc.List(context.Background(), "group", "project", testAttestationDigest, forge.ListAttestationOpts{})
			if err == nil {
				t.Fatalf("expected an error, got %d attestations", len(atts))
			}
			if tt.notFound {
				if !errors.Is(err, forge.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
				return
			}
			var errResp *gitlab.ErrorResponse
			if !errors.As(err, &errResp) || errResp.Response.StatusCode != tt.status {
				t.Errorf("expected a %d ErrorResponse, got %v", tt.status, err)
			}
		})
	}
}

func TestGitLabListAttestationsInvalidDigest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s", r.URL.Path)
	}))
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	_, err := f.Attestations().List(context.Background(), "group", "project", testAttestationHex, forge.ListAttestationOpts{})
	if !errors.Is(err, forge.ErrInvalidDigest) {
		t.Errorf("expected ErrInvalidDigest, got %v", err)
	}
}

func TestGitLabListThenDownloadAttestation(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+testAttestationPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, "[%s]", gitLabAttestationJSON(3, "provenance", "https://slsa.dev/provenance/v1"))
	})
	mux.HandleFunc("GET /api/v4/projects/group%2Fproject/attestations/3/download", func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "PRIVATE-TOKEN", "test-token", r.Header.Get("PRIVATE-TOKEN"))
		w.Header().Set("Content-Type", "text/json")
		_, _ = fmt.Fprint(w, testBundle)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	atts, err := f.Attestations().List(context.Background(), "group", "project", testAttestationDigest, forge.ListAttestationOpts{})
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}
	if len(atts) != 1 {
		t.Fatalf("expected 1 attestation, got %d", len(atts))
	}
	bundle, err := f.Attestations().Download(context.Background(), "group", "project", atts[0])
	if err != nil {
		t.Fatalf("Download: unexpected error: %v", err)
	}
	assertEqual(t, "bundle", testBundle, string(bundle))
}

func TestGitLabDownloadAttestationErrors(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		notFound bool
	}{
		{"not found", http.StatusNotFound, `{"message":"404 Not Found"}`, true},
		{"forbidden", http.StatusForbidden, `{"message":"403 Forbidden"}`, false},
		{"server error", http.StatusInternalServerError, `{"message":"500 Internal Server Error"}`, false},
		{"not json", http.StatusOK, `not json`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("GET /api/v4/projects/group%2Fproject/attestations/3/download", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprint(w, tt.body)
			})

			srv := httptest.NewServer(mux)
			defer srv.Close()

			svc := newTestGitLabAttestationService(t, srv)
			bundle, err := svc.Download(context.Background(), "group", "project", forge.Attestation{ID: "3"})
			if err == nil {
				t.Fatalf("expected an error, got bundle %s", bundle)
			}
			if tt.notFound != errors.Is(err, forge.ErrNotFound) {
				t.Errorf("errors.Is(err, ErrNotFound) = %v, want %v (err: %v)", !tt.notFound, tt.notFound, err)
			}
		})
	}
}

func TestGitLabDownloadAttestationInvalidID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s", r.URL.Path)
	}))
	defer srv.Close()

	f := New(srv.URL, "test-token", nil)
	for _, id := range []string{"", "0", "-1", "abc", "1/../2"} {
		if _, err := f.Attestations().Download(context.Background(), "group", "project", forge.Attestation{ID: id}); err == nil {
			t.Errorf("Download with ID %q: expected an error", id)
		}
	}
}

package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"

	forge "github.com/git-pkgs/forge"
)

const (
	bitbucketCommitSHA      = "8e8c483db84b4bee98b60c0593521ed34d9990e8"
	bitbucketCommitSHAUpper = "8E8C483DB84B4BEE98B60C0593521ED34D9990E8"
)

// newCommitTestForge points the Bitbucket API at a test server running h.
func newCommitTestForge(t *testing.T, h http.Handler) forge.Forge {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	t.Cleanup(func() { setBitbucketAPI(origAPI) })
	return New("test-token", srv.Client())
}

func writeRefTarget(w http.ResponseWriter, hash string) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"target": map[string]any{"hash": hash},
	})
}

func TestBitbucketResolveCommit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/commit/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want bearer token", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"hash": bitbucketCommitSHAUpper})
	})

	f := newCommitTestForge(t, mux)
	sha, err := f.Commits().ResolveCommit(context.Background(), "atlassian", "stash", "v1.0.0")
	if err != nil {
		t.Fatalf("ResolveCommit() error = %v", err)
	}
	if sha != bitbucketCommitSHA {
		t.Errorf("ResolveCommit() = %q, want %q", sha, bitbucketCommitSHA)
	}
}

func TestBitbucketResolveCommitSkipsRequestForFullSHA(t *testing.T) {
	var requests atomic.Int32
	f := newCommitTestForge(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))

	sha, err := f.Commits().ResolveCommit(context.Background(), "atlassian", "stash", bitbucketCommitSHAUpper)
	if err != nil {
		t.Fatalf("ResolveCommit() error = %v", err)
	}
	if sha != bitbucketCommitSHA {
		t.Errorf("ResolveCommit() = %q, want normalized %q", sha, bitbucketCommitSHA)
	}
	if got := requests.Load(); got != 0 {
		t.Errorf("ResolveCommit() made %d requests for a full SHA, want 0", got)
	}
}

func TestBitbucketResolveCommitNotFound(t *testing.T) {
	f := newCommitTestForge(t, http.NotFoundHandler())
	_, err := f.Commits().ResolveCommit(context.Background(), "atlassian", "stash", "missing")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("ResolveCommit() error = %v, want ErrNotFound", err)
	}
}

func TestBitbucketResolveCommitRejectsShortSHAInResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/commit/v1.0.0", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"hash": "8e8c483"})
	})

	f := newCommitTestForge(t, mux)
	if _, err := f.Commits().ResolveCommit(context.Background(), "atlassian", "stash", "v1.0.0"); err == nil {
		t.Fatal("ResolveCommit() error = nil, want invalid SHA error")
	}
}

func TestBitbucketResolveCommitRequiresArguments(t *testing.T) {
	f := New("", nil)
	if _, err := f.Commits().ResolveCommit(context.Background(), "", "stash", "v1.0.0"); !errors.Is(err, forge.ErrCommitRefRequired) {
		t.Errorf("ResolveCommit() error = %v, want ErrCommitRefRequired", err)
	}
}

func TestBitbucketResolveCommitSlashedRef(t *testing.T) {
	tests := []struct {
		name     string
		branches bool // whether the ref exists as a branch
		tags     bool // whether the ref exists as a tag
		wantSHA  string
		wantErr  error
		wantURLs []string
	}{
		{
			name:     "branch",
			branches: true,
			tags:     true,
			wantSHA:  bitbucketCommitSHA,
			wantURLs: []string{"/2.0/repositories/atlassian/stash/refs/branches/feature%2Flogin"},
		},
		{
			name:    "tag",
			tags:    true,
			wantSHA: bitbucketCommitSHA,
			wantURLs: []string{
				"/2.0/repositories/atlassian/stash/refs/branches/feature%2Flogin",
				"/2.0/repositories/atlassian/stash/refs/tags/feature%2Flogin",
			},
		},
		{
			name:    "missing",
			wantErr: forge.ErrNotFound,
			wantURLs: []string{
				"/2.0/repositories/atlassian/stash/refs/branches/feature%2Flogin",
				"/2.0/repositories/atlassian/stash/refs/tags/feature%2Flogin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			f := newCommitTestForge(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = append(got, r.URL.EscapedPath())
				switch r.URL.EscapedPath() {
				case "/2.0/repositories/atlassian/stash/refs/branches/feature%2Flogin":
					if tt.branches {
						writeRefTarget(w, bitbucketCommitSHA)
						return
					}
				case "/2.0/repositories/atlassian/stash/refs/tags/feature%2Flogin":
					if tt.tags {
						writeRefTarget(w, bitbucketCommitSHA)
						return
					}
				}
				http.NotFound(w, r)
			}))

			sha, err := f.Commits().ResolveCommit(context.Background(), "atlassian", "stash", "feature/login")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ResolveCommit() error = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("ResolveCommit() error = %v", err)
			}
			if sha != tt.wantSHA {
				t.Errorf("ResolveCommit() = %q, want %q", sha, tt.wantSHA)
			}
			if !slices.Equal(got, tt.wantURLs) {
				t.Errorf("requested %v, want %v", got, tt.wantURLs)
			}
		})
	}
}

func TestBitbucketResolveCommitSlashedRefServerError(t *testing.T) {
	f := newCommitTestForge(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	_, err := f.Commits().ResolveCommit(context.Background(), "atlassian", "stash", "feature/login")
	if !errors.Is(err, forge.ErrServerError) {
		t.Errorf("ResolveCommit() error = %v, want ErrServerError", err)
	}
}

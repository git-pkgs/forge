//go:build !tinygo

package gitlab

import (
	"context"
	"errors"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func assertSentinel(t *testing.T, err, want error) {
	t.Helper()
	for _, sentinel := range []error{forge.ErrUnauthorized, forge.ErrRateLimited, forge.ErrServerError} {
		if got := errors.Is(err, sentinel); got != (sentinel == want) {
			t.Errorf("errors.Is(%v, %v) = %v", err, sentinel, got)
		}
	}
}

func errorForge(t *testing.T, status int, body string) forge.Forge {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return &gitLabForge{client: noRetryClient(t, srv)}
}

// noRetryClient builds a client that returns 429 and 5xx responses straight
// away. By default the GitLab client retries them for several seconds.
func noRetryClient(t *testing.T, srv *httptest.Server) *gitlab.Client {
	t.Helper()
	c, err := gitlab.NewClient("token",
		gitlab.WithBaseURL(srv.URL),
		gitlab.WithHTTPClient(srv.Client()),
		gitlab.WithCustomRetryMax(0),
	)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestGitLabErrorsAreClassified(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   error
	}{
		{"unauthorized", http.StatusUnauthorized, forge.ErrUnauthorized},
		{"forbidden", http.StatusForbidden, forge.ErrUnauthorized},
		{"too many requests", http.StatusTooManyRequests, forge.ErrRateLimited},
		{"server error", http.StatusInternalServerError, forge.ErrServerError},
		{"validation failure", http.StatusUnprocessableEntity, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := errorForge(t, tt.status, `{"message":"failure"}`)
			_, err := f.Repos().Get(context.Background(), "owner", "repo")
			if err == nil {
				t.Fatal("expected an error")
			}
			assertSentinel(t, err, tt.want)
		})
	}
}

func TestGitLabErrorsAreClassifiedAcrossServices(t *testing.T) {
	ctx := context.Background()
	calls := map[string]func(f forge.Forge) error{
		"issues": func(f forge.Forge) error {
			_, err := f.Issues().Get(ctx, "owner", "repo", 1)
			return err
		},
		"merge requests": func(f forge.Forge) error {
			_, err := f.PullRequests().List(ctx, "owner", "repo", forge.ListPROpts{})
			return err
		},
		"labels": func(f forge.Forge) error {
			_, err := f.Labels().List(ctx, "owner", "repo", forge.ListLabelOpts{})
			return err
		},
		"releases": func(f forge.Forge) error {
			_, err := f.Releases().List(ctx, "owner", "repo", forge.ListReleaseOpts{})
			return err
		},
		"branches": func(f forge.Forge) error {
			_, err := f.Branches().List(ctx, "owner", "repo", forge.ListBranchOpts{})
			return err
		},
		"authenticated user": func(f forge.Forge) error {
			_, err := f.GetAuthenticatedUser(ctx)
			return err
		},
		"user lookup wrapped with context": func(f forge.Forge) error {
			_, err := f.Issues().Create(ctx, "owner", "repo", forge.CreateIssueOpts{Title: "t", Assignees: []string{"alice"}})
			return err
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			f := errorForge(t, http.StatusUnauthorized, `{"message":"401 Unauthorized"}`)
			assertSentinel(t, call(f), forge.ErrUnauthorized)
		})
	}
}

func TestGitLabSecretCreateErrorIsClassified(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	f := &gitLabForge{client: noRetryClient(t, srv)}
	err := f.Secrets().Set(context.Background(), "owner", "repo", forge.SetSecretOpts{Name: "NAME", Value: "value"})
	assertSentinel(t, err, forge.ErrUnauthorized)
}

func TestGitLabClassifiedErrorKeepsDetails(t *testing.T) {
	f := errorForge(t, http.StatusForbidden, `{"message":"403 Forbidden"}`)
	_, err := f.Repos().Get(context.Background(), "owner", "repo")

	var errResp *gitlab.ErrorResponse
	if !errors.As(err, &errResp) {
		t.Fatalf("errors.As should find the client-go error in %v", err)
	}
	if err.Error() != errResp.Error() {
		t.Errorf("message = %q, want %q", err.Error(), errResp.Error())
	}
}

func TestGitLabWrapErrPassesOtherErrorsThrough(t *testing.T) {
	if err := wrapErr(nil); err != nil {
		t.Errorf("wrapErr(nil) = %v", err)
	}
	original := errors.New("connection refused")
	if err := wrapErr(original); err != original {
		t.Errorf("wrapErr(%v) = %v, want it unchanged", original, err)
	}
}

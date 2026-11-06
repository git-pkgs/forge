package github

import (
	"context"
	"errors"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v82/github"
)

func assertSentinel(t *testing.T, err, want error) {
	t.Helper()
	for _, sentinel := range []error{forge.ErrUnauthorized, forge.ErrRateLimited, forge.ErrServerError} {
		if got := errors.Is(err, sentinel); got != (sentinel == want) {
			t.Errorf("errors.Is(%v, %v) = %v", err, sentinel, got)
		}
	}
}

func errorForge(t *testing.T, status int, header http.Header, body string) forge.Forge {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for key, values := range header {
			w.Header()[key] = values
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewWithBase(srv.URL, "", srv.Client())
}

func TestGitHubErrorsAreClassified(t *testing.T) {
	tests := []struct {
		name   string
		status int
		header http.Header
		body   string
		want   error
	}{
		{name: "bad credentials", status: http.StatusUnauthorized, body: `{"message":"Bad credentials"}`, want: forge.ErrUnauthorized},
		{name: "two factor required", status: http.StatusUnauthorized, header: http.Header{"X-Github-Otp": {"required; app"}}, want: forge.ErrUnauthorized},
		{name: "forbidden", status: http.StatusForbidden, body: `{"message":"Must have admin rights"}`, want: forge.ErrUnauthorized},
		{name: "primary rate limit", status: http.StatusForbidden, header: http.Header{"X-Ratelimit-Remaining": {"0"}}, body: `{"message":"API rate limit exceeded"}`, want: forge.ErrRateLimited},
		{name: "secondary rate limit", status: http.StatusForbidden, body: `{"message":"slow down","documentation_url":"https://docs.github.com/rest/overview/rate-limits-for-the-rest-api#about-secondary-rate-limits"}`, want: forge.ErrRateLimited},
		{name: "too many requests", status: http.StatusTooManyRequests, want: forge.ErrRateLimited},
		{name: "server error", status: http.StatusInternalServerError, want: forge.ErrServerError},
		{name: "bad gateway", status: http.StatusBadGateway, body: "<html>bad gateway</html>", want: forge.ErrServerError},
		{name: "validation failure", status: http.StatusUnprocessableEntity, body: `{"message":"Validation Failed"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := errorForge(t, tt.status, tt.header, tt.body)
			_, err := f.Repos().Get(context.Background(), "owner", "repo")
			if err == nil {
				t.Fatal("expected an error")
			}
			assertSentinel(t, err, tt.want)
		})
	}
}

func TestGitHubErrorsAreClassifiedAcrossServices(t *testing.T) {
	ctx := context.Background()
	calls := map[string]func(f forge.Forge) error{
		"issues": func(f forge.Forge) error {
			_, err := f.Issues().Get(ctx, "owner", "repo", 1)
			return err
		},
		"pull requests": func(f forge.Forge) error {
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
		"rate limit": func(f forge.Forge) error {
			_, err := f.GetRateLimit(ctx)
			return err
		},
		"close pull request": func(f forge.Forge) error {
			return f.PullRequests().Close(ctx, "owner", "repo", 1)
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			f := errorForge(t, http.StatusUnauthorized, nil, `{"message":"Bad credentials"}`)
			assertSentinel(t, call(f), forge.ErrUnauthorized)
		})
	}
}

func TestGitHubClassifiedErrorKeepsDetails(t *testing.T) {
	f := errorForge(t, http.StatusForbidden, nil, `{"message":"Must have admin rights"}`)
	_, err := f.Repos().Get(context.Background(), "owner", "repo")

	var errResp *github.ErrorResponse
	if !errors.As(err, &errResp) {
		t.Fatalf("errors.As should find the go-github error in %v", err)
	}
	if err.Error() != errResp.Error() {
		t.Errorf("message = %q, want %q", err.Error(), errResp.Error())
	}
}

func TestGitHubWrapErrPassesOtherErrorsThrough(t *testing.T) {
	if err := wrapErr(nil); err != nil {
		t.Errorf("wrapErr(nil) = %v", err)
	}
	original := errors.New("connection refused")
	if err := wrapErr(original); err != original {
		t.Errorf("wrapErr(%v) = %v, want it unchanged", original, err)
	}
}

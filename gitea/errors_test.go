package gitea

import (
	"context"
	"errors"
	forge "github.com/git-pkgs/forge"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.gitea.io/sdk/gitea"
)

func newResp(code int, status string) *gitea.Response {
	return &gitea.Response{Response: &http.Response{StatusCode: code, Status: status}}
}

func TestWrapErr(t *testing.T) {
	tests := []struct {
		name string
		op   string
		resp *gitea.Response
		err  error
		want string
	}{
		{"blank message", "create issue", newResp(500, "500 Internal Server Error"), errors.New(""), "create issue: 500 Internal Server Error"},
		{"whitespace message", "create issue", newResp(500, "500 Internal Server Error"), errors.New("  \n"), "create issue: 500 Internal Server Error"},
		{"with message", "create issue", newResp(422, "422 Unprocessable Entity"), errors.New("bad input"), "create issue: 422 Unprocessable Entity: bad input"},
		{"nil resp with message", "list labels", nil, errors.New("dial tcp: connection refused"), "list labels: dial tcp: connection refused"},
		{"nil resp blank message", "get issue", nil, errors.New(""), "get issue"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := wrapErr(tc.op, tc.resp, tc.err)
			if got.Error() != tc.want {
				t.Errorf("want %q, got %q", tc.want, got.Error())
			}
			if !errors.Is(got, tc.err) {
				t.Errorf("wrapped error should unwrap to the original, got %v", got)
			}
		})
	}
}

func TestWrapErrNotFound(t *testing.T) {
	got := wrapErr("get issue", newResp(404, "404 Not Found"), errors.New("The target couldn't be found."))
	if !errors.Is(got, forge.ErrNotFound) {
		t.Errorf("404 should return ErrNotFound sentinel, got %v", got)
	}
}

// Transport failures come back with a nil response and an error the caller may
// want to inspect, so the chain has to survive wrapping.
func TestWrapErrUnwrapsTransportError(t *testing.T) {
	inner := errors.New("connection refused")
	got := wrapErr("list releases", nil, &net.OpError{Op: "dial", Err: inner})

	var opErr *net.OpError
	if !errors.As(got, &opErr) {
		t.Fatalf("want *net.OpError in the chain, got %v", got)
	}
	if !errors.Is(got, inner) {
		t.Errorf("want the innermost error in the chain, got %v", got)
	}
}

func assertSentinel(t *testing.T, err, want error) {
	t.Helper()
	for _, sentinel := range []error{forge.ErrUnauthorized, forge.ErrRateLimited, forge.ErrServerError} {
		if got := errors.Is(err, sentinel); got != (sentinel == want) {
			t.Errorf("errors.Is(%v, %v) = %v", err, sentinel, got)
		}
	}
}

func TestGiteaErrorsAreClassified(t *testing.T) {
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
	ctx := context.Background()
	calls := map[string]func(f forge.Forge) error{
		"repos": func(f forge.Forge) error {
			_, err := f.Repos().Get(ctx, "owner", "repo")
			return err
		},
		"issues": func(f forge.Forge) error {
			_, err := f.Issues().Get(ctx, "owner", "repo", 1)
			return err
		},
		"pull requests": func(f forge.Forge) error {
			_, err := f.PullRequests().Get(ctx, "owner", "repo", 1)
			return err
		},
		"releases": func(f forge.Forge) error {
			_, err := f.Releases().List(ctx, "owner", "repo", forge.ListReleaseOpts{})
			return err
		},
		"labels resolved with context": func(f forge.Forge) error {
			_, err := f.Issues().Create(ctx, "owner", "repo", forge.CreateIssueOpts{Title: "t", Labels: []string{"bug"}})
			return err
		},
		"rate limit": func(f forge.Forge) error {
			_, err := f.GetRateLimit(ctx)
			return err
		},
	}

	for _, tt := range tests {
		for name, call := range calls {
			t.Run(tt.name+"/"+name, func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.status)
					_, _ = w.Write([]byte(`{"message":"failure"}`))
				}))
				defer srv.Close()

				err := call(New(srv.URL, "token", srv.Client()))
				if err == nil {
					t.Fatal("expected an error")
				}
				assertSentinel(t, err, tt.want)
			})
		}
	}
}

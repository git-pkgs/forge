package gitea

import (
	"errors"
	forge "github.com/git-pkgs/forge"
	"net"
	"net/http"
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

package gitea

import (
	"errors"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"testing"

	"code.gitea.io/sdk/gitea"
)

func resp(code int, status string) *gitea.Response {
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
		{"blank message", "create issue", resp(500, "500 Internal Server Error"), errors.New(""), "create issue: 500 Internal Server Error"},
		{"whitespace message", "create issue", resp(500, "500 Internal Server Error"), errors.New("  \n"), "create issue: 500 Internal Server Error"},
		{"with message", "create issue", resp(422, "422 Unprocessable Entity"), errors.New("bad input"), "create issue: 422 Unprocessable Entity: bad input"},
		{"nil resp with message", "list labels", nil, errors.New("dial tcp: connection refused"), "list labels: dial tcp: connection refused"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := wrapErr(tc.op, tc.resp, tc.err)
			if got.Error() != tc.want {
				t.Errorf("want %q, got %q", tc.want, got.Error())
			}
		})
	}
}

func TestWrapErrNotFound(t *testing.T) {
	got := wrapErr("get issue", resp(404, "404 Not Found"), errors.New("The target couldn't be found."))
	if !errors.Is(got, forge.ErrNotFound) {
		t.Errorf("404 should return ErrNotFound sentinel, got %v", got)
	}
}

func TestWrapErrNilRespBlankMessage(t *testing.T) {
	inner := errors.New("")
	got := wrapErr("get issue", nil, inner)
	if got.Error() != "get issue: " {
		t.Errorf("want %q, got %q", "get issue: ", got.Error())
	}
	if !errors.Is(got, inner) {
		t.Error("nil-resp blank error should wrap the original for errors.Is")
	}
}

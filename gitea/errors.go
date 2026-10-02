package gitea

import (
	forge "github.com/git-pkgs/forge"
	"net/http"
	"strings"

	"code.gitea.io/sdk/gitea"
)

type apiError struct {
	op     string
	status string
	msg    string
	err    error
}

func (e *apiError) Error() string {
	parts := []string{e.op}
	if e.status != "" {
		parts = append(parts, e.status)
	}
	if e.msg != "" {
		parts = append(parts, e.msg)
	}
	return strings.Join(parts, ": ")
}

func (e *apiError) Unwrap() error { return e.err }

// wrapErr adds the operation name and HTTP status to an SDK error. The SDK
// returns errors containing only the server's "message" field, which Forgejo
// blanks on 500s in production mode and which can otherwise be cryptic
// (e.g. "Release has no Tag" for a 409). Without the status code the caller
// has nothing to go on.
func wrapErr(op string, resp *gitea.Response, err error) error {
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return forge.ErrNotFound
	}
	e := &apiError{op: op, msg: strings.TrimSpace(err.Error()), err: err}
	if resp != nil {
		e.status = resp.Status
	}
	return e
}

package gitea

import (
	"fmt"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"strings"

	"code.gitea.io/sdk/gitea"
)

// wrapErr adds the operation name and HTTP status to an SDK error. The SDK
// returns errors containing only the server's "message" field, which Forgejo
// blanks on 500s in production mode and which can otherwise be cryptic
// (e.g. "Release has no Tag" for a 409). Without the status code the caller
// has nothing to go on.
func wrapErr(op string, resp *gitea.Response, err error) error {
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return forge.ErrNotFound
	}
	msg := strings.TrimSpace(err.Error())
	if resp == nil {
		if msg == "" {
			return fmt.Errorf("%s: %w", op, err)
		}
		return fmt.Errorf("%s: %s", op, msg)
	}
	if msg == "" {
		return fmt.Errorf("%s: %s", op, resp.Status)
	}
	return fmt.Errorf("%s: %s: %s", op, resp.Status, msg)
}

//go:build !tinygo

package gitlab

import (
	"errors"
	forge "github.com/git-pkgs/forge"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// wrapErr classifies an error from the GitLab client so callers can match it
// with errors.Is, while keeping its message and its client-go error type.
func wrapErr(err error) error {
	var errResp *gitlab.ErrorResponse
	if errors.As(err, &errResp) && errResp.Response != nil {
		return forge.ClassifyStatus(errResp.Response.StatusCode, err)
	}
	return err
}

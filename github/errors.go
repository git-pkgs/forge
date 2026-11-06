package github

import (
	"errors"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"github.com/google/go-github/v82/github"
)

// wrapErr classifies an error from the GitHub client so callers can match it
// with errors.Is, while keeping its message and its go-github error type.
func wrapErr(err error) error {
	// GitHub reports rate limits as 403 as well as 429, and go-github gives
	// those their own types. Report them all as a 429 so they are not mistaken
	// for an authorization failure.
	var rateLimit *github.RateLimitError
	var abuseRateLimit *github.AbuseRateLimitError
	if errors.As(err, &rateLimit) || errors.As(err, &abuseRateLimit) {
		return forge.ClassifyStatus(http.StatusTooManyRequests, err)
	}

	var twoFactor *github.TwoFactorAuthError
	if errors.As(err, &twoFactor) {
		return forge.ClassifyStatus(http.StatusUnauthorized, err)
	}

	var errResp *github.ErrorResponse
	if errors.As(err, &errResp) && errResp.Response != nil {
		return forge.ClassifyStatus(errResp.Response.StatusCode, err)
	}
	return err
}

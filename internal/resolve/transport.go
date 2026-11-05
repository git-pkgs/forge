package resolve

import (
	"errors"
	"fmt"
	"net/http"
)

// maxRedirects matches the limit net/http applies when CheckRedirect is nil.
const maxRedirects = 10

var userAgent = "forge/dev"

// SetUserAgent sets the User-Agent string sent on every HTTP request made
// through HTTPClient. The CLI calls this at startup with the build version.
func SetUserAgent(ua string) {
	userAgent = ua
}

// HTTPClient returns an http.Client whose transport sets the User-Agent
// header on outbound requests. Used for all forge API traffic so requests
// are identifiable in server logs. It refuses redirects from HTTPS to HTTP,
// since net/http would otherwise resend headers such as Authorization in
// cleartext when the redirect stays on the same host.
func HTTPClient() *http.Client {
	return &http.Client{
		Transport:     &userAgentTransport{base: http.DefaultTransport},
		CheckRedirect: checkRedirect,
	}
}

// checkRedirect keeps the default redirect limit and rejects any hop that
// downgrades from HTTPS to HTTP.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return errors.New("stopped after 10 redirects")
	}
	if len(via) > 0 && via[len(via)-1].URL.Scheme == "https" && req.URL.Scheme == "http" {
		return fmt.Errorf("refusing redirect from %s to insecure %s", via[len(via)-1].URL.Redacted(), req.URL.Redacted())
	}
	return nil
}

type userAgentTransport struct {
	base http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") != "" {
		return t.base.RoundTrip(req)
	}
	r := req.Clone(req.Context())
	r.Header.Set("User-Agent", userAgent)
	return t.base.RoundTrip(r)
}

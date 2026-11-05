package resolve

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
		return fmt.Errorf("refusing redirect from %s to insecure %s", urlForError(via[len(via)-1].URL), urlForError(req.URL))
	}
	return nil
}

// urlForError formats u for an error message without userinfo, query or
// fragment, any of which can carry credentials such as an access_token
// parameter or a signed URL.
func urlForError(u *url.URL) string {
	c := *u
	c.User = nil
	c.RawQuery = ""
	c.ForceQuery = false
	c.Fragment = ""
	c.RawFragment = ""
	return c.String()
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

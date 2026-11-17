package forges

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Credentials authenticates requests to a forge. Authenticate adds whatever
// the forge expects, such as an Authorization header or a request signature,
// to req. It is called for every request a client returned by AuthHTTPClient
// sends, including redirects, with a copy of the request that it may modify.
// The copy shares the original's body, so an implementation that signs the
// body should read it from req.GetBody. An error stops the request from being
// sent.
//
// Requests to every host pass through Authenticate, including detection
// probes, redirects and downloads from other hosts, so implementations should
// check req.URL before adding anything.
type Credentials interface {
	Authenticate(req *http.Request) error
}

// CredentialsFunc adapts a function to Credentials.
type CredentialsFunc func(req *http.Request) error

// Authenticate calls f(req).
func (f CredentialsFunc) Authenticate(req *http.Request) error {
	return f(req)
}

// TokenSource supplies the token for a request. Token is called for every
// request, so it can cache a short-lived token, such as a GitHub App
// installation token, and refresh it when it expires.
type TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// TokenSourceFunc adapts a function to TokenSource.
type TokenSourceFunc func(ctx context.Context) (string, error)

// Token calls f(ctx).
func (f TokenSourceFunc) Token(ctx context.Context) (string, error) {
	return f(ctx)
}

// BearerToken returns Credentials that send the token from ts as
// "Authorization: Bearer <token>" on requests to host and its subdomains, so
// "github.com" also covers api.github.com. host may also be a URL such as
// https://git.example.com. Hosts are compared without their ports and
// case-insensitively. Requests to any other host are sent without
// the token. An empty token sends the request without credentials.
//
// GitHub, GitLab, Gitea/Forgejo, Bitbucket Cloud and Tangled accept bearer
// tokens.
func BearerToken(host string, ts TokenSource) Credentials {
	domain := strings.ToLower(hostname(host))
	return CredentialsFunc(func(req *http.Request) error {
		if !hostWithin(req.URL.Hostname(), domain) {
			return nil
		}
		token, err := ts.Token(req.Context())
		if err != nil {
			return fmt.Errorf("token for %s: %w", host, err)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		return nil
	})
}

// hostname returns the host name in host, which may be a bare host, a host
// and port, or a URL.
func hostname(host string) string {
	if strings.Contains(host, "://") {
		if u, err := url.Parse(host); err == nil {
			return u.Hostname()
		}
		return ""
	}
	return (&url.URL{Host: host}).Hostname()
}

// hostWithin reports whether host is domain or one of its subdomains.
func hostWithin(host, domain string) bool {
	if domain == "" {
		return false
	}
	host = strings.ToLower(host)
	return host == domain || strings.HasSuffix(host, "."+domain)
}

// AuthHTTPClient returns a copy of hc that passes every request to creds
// before sending it. A nil hc copies http.DefaultClient.
//
// Pass the result to each backend constructor with an empty token, so the
// backend adds no credentials of its own, and to WithHTTPClient so forges
// registered by RegisterDomain use it too. Gerrit only uses its authenticated
// endpoints when it has a token, so give gerrit.New the token instead.
func AuthHTTPClient(hc *http.Client, creds Credentials) *http.Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	c := *hc
	base := c.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	c.Transport = &authTransport{base: base, creds: creds}
	return &c
}

type authTransport struct {
	base  http.RoundTripper
	creds Credentials
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	if err := t.creds.Authenticate(r); err != nil {
		if req.Body != nil {
			_ = req.Body.Close()
		}
		return nil, err
	}
	return t.base.RoundTrip(r)
}

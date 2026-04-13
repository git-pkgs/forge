package resolve

import "net/http"

var userAgent = "forge/dev"

// SetUserAgent sets the User-Agent string sent on every HTTP request made
// through HTTPClient. The CLI calls this at startup with the build version.
func SetUserAgent(ua string) {
	userAgent = ua
}

// HTTPClient returns an http.Client whose transport sets the User-Agent
// header on outbound requests. Used for all forge API traffic so requests
// are identifiable in server logs.
func HTTPClient() *http.Client {
	return &http.Client{Transport: &userAgentTransport{base: http.DefaultTransport}}
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

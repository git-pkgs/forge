package forges

import "net/http"

// ReadOnlyHTTPClient returns a copy of hc that only sends GET, HEAD and
// OPTIONS requests. Any other request fails with ErrReadOnly before anything
// is sent. A nil hc copies http.DefaultClient.
//
// Pass the result to each backend constructor, and to WithHTTPClient so
// forges registered by RegisterDomain use it too.
func ReadOnlyHTTPClient(hc *http.Client) *http.Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	c := *hc
	base := c.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	c.Transport = &readOnlyTransport{base: base}
	return &c
}

type readOnlyTransport struct {
	base http.RoundTripper
}

func (t *readOnlyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.Method {
	case "", http.MethodGet, http.MethodHead, http.MethodOptions:
		return t.base.RoundTrip(req)
	}
	if req.Body != nil {
		_ = req.Body.Close()
	}
	return nil, ErrReadOnly
}

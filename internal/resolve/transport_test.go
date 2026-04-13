package resolve

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPClientSetsUserAgent(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
	}))
	defer srv.Close()

	c := HTTPClient()
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if got != userAgent {
		t.Errorf("expected User-Agent %q, got %q", userAgent, got)
	}
	if got == "" {
		t.Error("User-Agent was empty")
	}
}

func TestSetUserAgent(t *testing.T) {
	old := userAgent
	defer func() { userAgent = old }()

	SetUserAgent("forge/1.2.3")

	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
	}))
	defer srv.Close()

	c := HTTPClient()
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if got != "forge/1.2.3" {
		t.Errorf("expected User-Agent forge/1.2.3, got %q", got)
	}
}

func TestUserAgentTransportPreservesExisting(t *testing.T) {
	// If a caller has already set User-Agent (e.g. forge api -H "User-Agent: x"),
	// the transport must not stomp on it.
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
	}))
	defer srv.Close()

	c := HTTPClient()
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	req.Header.Set("User-Agent", "custom/1.0")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if got != "custom/1.0" {
		t.Errorf("expected explicit User-Agent to be preserved, got %q", got)
	}
}

func TestUserAgentTransportDoesNotMutateRequest(t *testing.T) {
	// RoundTrippers must not modify the original request.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	c := HTTPClient()
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if req.Header.Get("User-Agent") != "" {
		t.Errorf("transport mutated the original request: User-Agent=%q", req.Header.Get("User-Agent"))
	}
}

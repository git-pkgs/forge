package resolve

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

// tlsHTTPClient returns HTTPClient with its transport trusting srv's
// self-signed certificate, keeping the redirect policy under test.
func tlsHTTPClient(srv *httptest.Server) *http.Client {
	c := HTTPClient()
	c.Transport = &userAgentTransport{base: srv.Client().Transport}
	return c
}

func TestHTTPClientRejectsHTTPSToHTTPRedirect(t *testing.T) {
	var plainHits int
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		plainHits++
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Authorization header sent over plain HTTP")
		}
	}))
	defer plain.Close()

	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+"/target", http.StatusFound)
	}))
	defer secure.Close()

	req, _ := http.NewRequest(http.MethodGet, secure.URL, nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp, err := tlsHTTPClient(secure).Do(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected HTTPS to HTTP redirect to be refused")
	}
	if !strings.Contains(err.Error(), "refusing redirect") {
		t.Errorf("unexpected error: %v", err)
	}
	if plainHits != 0 {
		t.Errorf("plain HTTP server was contacted %d times", plainHits)
	}
}

func TestHTTPClientFollowsSafeRedirects(t *testing.T) {
	t.Run("https to https", func(t *testing.T) {
		var srv *httptest.Server
		srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/start" {
				http.Redirect(w, r, srv.URL+"/end", http.StatusFound)
				return
			}
			_, _ = w.Write([]byte("ok"))
		}))
		defer srv.Close()

		resp, err := tlsHTTPClient(srv).Get(srv.URL + "/start")
		if err != nil {
			t.Fatalf("redirect within HTTPS failed: %v", err)
		}
		_ = resp.Body.Close()
		if resp.Request.URL.Path != "/end" {
			t.Errorf("final path = %q, want /end", resp.Request.URL.Path)
		}
	})

	t.Run("http to https", func(t *testing.T) {
		secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		defer secure.Close()
		plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, secure.URL+"/end", http.StatusFound)
		}))
		defer plain.Close()

		resp, err := tlsHTTPClient(secure).Get(plain.URL)
		if err != nil {
			t.Fatalf("upgrade redirect failed: %v", err)
		}
		_ = resp.Body.Close()
		if resp.Request.URL.Scheme != "https" {
			t.Errorf("final scheme = %q, want https", resp.Request.URL.Scheme)
		}
	})

	t.Run("http to http", func(t *testing.T) {
		var srv *httptest.Server
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/start" {
				http.Redirect(w, r, srv.URL+"/end", http.StatusFound)
			}
		}))
		defer srv.Close()

		resp, err := HTTPClient().Get(srv.URL + "/start")
		if err != nil {
			t.Fatalf("redirect within HTTP failed: %v", err)
		}
		_ = resp.Body.Close()
		if resp.Request.URL.Path != "/end" {
			t.Errorf("final path = %q, want /end", resp.Request.URL.Path)
		}
	})
}

func TestHTTPClientKeepsRedirectLimit(t *testing.T) {
	var hits int
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Redirect(w, r, srv.URL+"/loop", http.StatusFound)
	}))
	defer srv.Close()

	resp, err := HTTPClient().Get(srv.URL)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected redirect loop to stop")
	}
	if !strings.Contains(err.Error(), "stopped after 10 redirects") {
		t.Errorf("unexpected error: %v", err)
	}
	if hits != maxRedirects {
		t.Errorf("server hit %d times, want %d", hits, maxRedirects)
	}
}

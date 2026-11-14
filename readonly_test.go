package forges

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestReadOnlyHTTPClientAllowsReads(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Method)
	}))
	defer srv.Close()

	hc := ReadOnlyHTTPClient(srv.Client())
	for _, method := range []string{"", http.MethodGet, http.MethodHead, http.MethodOptions} {
		req, err := http.NewRequest(method, srv.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := hc.Do(req)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", method, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%q: status = %d, want 200", method, resp.StatusCode)
		}
	}
}

type closeRecorder struct {
	io.Reader
	closed bool
}

func (c *closeRecorder) Close() error {
	c.closed = true
	return nil
}

func TestReadOnlyHTTPClientRefusesWrites(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
	}))
	defer srv.Close()

	hc := ReadOnlyHTTPClient(srv.Client())
	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodTrace, "PROPFIND"}
	for _, method := range methods {
		body := &closeRecorder{Reader: strings.NewReader(`{"title":"x"}`)}
		req, err := http.NewRequest(method, srv.URL+"/repos/o/r/issues", body)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := hc.Do(req)
		if resp != nil {
			_ = resp.Body.Close()
		}
		if !errors.Is(err, ErrReadOnly) {
			t.Errorf("%s: err = %v, want ErrReadOnly", method, err)
		}
		if !body.closed {
			t.Errorf("%s: request body was not closed", method)
		}
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("server received %d requests, want 0", n)
	}
}

func TestReadOnlyHTTPClientCopiesClient(t *testing.T) {
	base := &http.Client{Timeout: time.Minute}
	hc := ReadOnlyHTTPClient(base)

	if hc == base {
		t.Fatal("ReadOnlyHTTPClient returned the client it was given")
	}
	if base.Transport != nil {
		t.Errorf("original client's transport was changed to %T", base.Transport)
	}
	if hc.Timeout != time.Minute {
		t.Errorf("Timeout = %v, want %v", hc.Timeout, time.Minute)
	}
}

func TestReadOnlyHTTPClientNil(t *testing.T) {
	hc := ReadOnlyHTTPClient(nil)
	if hc == http.DefaultClient {
		t.Fatal("ReadOnlyHTTPClient(nil) returned http.DefaultClient")
	}
	if http.DefaultClient.Transport != nil {
		t.Errorf("http.DefaultClient's transport was changed to %T", http.DefaultClient.Transport)
	}

	_, err := hc.Post("http://127.0.0.1:0/", "application/json", strings.NewReader("{}"))
	if !errors.Is(err, ErrReadOnly) {
		t.Errorf("err = %v, want ErrReadOnly", err)
	}
}

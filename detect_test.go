package forges

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

type errTransport struct{ err error }

func (t errTransport) RoundTrip(*http.Request) (*http.Response, error) { return nil, t.err }

func TestDetectForgeTypeSurfacesTransportError(t *testing.T) {
	netErr := errors.New("dial tcp: lookup forge.invalid: no such host")
	hc := &http.Client{Transport: errTransport{err: netErr}}

	_, err := DetectForgeType(context.Background(), "forge.invalid", hc)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), netErr.Error()) {
		t.Fatalf("expected transport error to be surfaced, got: %v", err)
	}
}

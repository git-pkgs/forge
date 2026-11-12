package forges

import (
	"errors"
	"net/http"
	"testing"
)

var statusSentinels = []error{ErrUnauthorized, ErrRateLimited, ErrServerError}

func assertSentinel(t *testing.T, err, want error) {
	t.Helper()
	for _, sentinel := range statusSentinels {
		if got := errors.Is(err, sentinel); got != (sentinel == want) {
			t.Errorf("errors.Is(%v, %v) = %v", err, sentinel, got)
		}
	}
}

var statusTests = []struct {
	status int
	want   error
}{
	{http.StatusBadRequest, nil},
	{http.StatusUnauthorized, ErrUnauthorized},
	{http.StatusForbidden, ErrUnauthorized},
	{http.StatusNotFound, nil},
	{http.StatusConflict, nil},
	{http.StatusTooManyRequests, ErrRateLimited},
	{http.StatusInternalServerError, ErrServerError},
	{http.StatusBadGateway, ErrServerError},
	{599, ErrServerError},
	{600, nil},
}

func TestClassifyStatus(t *testing.T) {
	for _, tt := range statusTests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			original := errors.New("original message")
			got := ClassifyStatus(tt.status, original)
			assertSentinel(t, got, tt.want)
			if !errors.Is(got, original) {
				t.Errorf("result should wrap the original error, got %v", got)
			}
			if got.Error() != original.Error() {
				t.Errorf("message = %q, want %q", got.Error(), original.Error())
			}
		})
	}
}

func TestClassifyStatusNilError(t *testing.T) {
	if err := ClassifyStatus(http.StatusInternalServerError, nil); err != nil {
		t.Errorf("ClassifyStatus(500, nil) = %v, want nil", err)
	}
}

func TestClassifyStatusKeepsErrorTypes(t *testing.T) {
	var httpErr *HTTPError
	err := ClassifyStatus(http.StatusForbidden, &HTTPError{StatusCode: http.StatusForbidden})
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusForbidden {
		t.Errorf("errors.As should find the HTTPError in %v", err)
	}
}

func TestHTTPErrorIs(t *testing.T) {
	for _, tt := range statusTests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			assertSentinel(t, &HTTPError{StatusCode: tt.status, URL: "https://example.com"}, tt.want)
		})
	}
}

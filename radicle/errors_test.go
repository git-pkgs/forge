package radicle

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/git-pkgs/forge"
)

func TestRadicleErrorsAreClassified(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   error
	}{
		{"unauthorized", http.StatusUnauthorized, forges.ErrUnauthorized},
		{"forbidden", http.StatusForbidden, forges.ErrUnauthorized},
		{"too many requests", http.StatusTooManyRequests, forges.ErrRateLimited},
		{"server error", http.StatusInternalServerError, forges.ErrServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			f := New(srv.URL, "", srv.Client())
			_, err := f.Repos().Get(context.Background(), "", testRID)
			if !errors.Is(err, tt.want) {
				t.Errorf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

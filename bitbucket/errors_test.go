package bitbucket

import (
	"context"
	"errors"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBitbucketErrorsAreClassified(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   error
	}{
		{"unauthorized", http.StatusUnauthorized, forge.ErrUnauthorized},
		{"forbidden", http.StatusForbidden, forge.ErrUnauthorized},
		{"too many requests", http.StatusTooManyRequests, forge.ErrRateLimited},
		{"server error", http.StatusInternalServerError, forge.ErrServerError},
	}
	ctx := context.Background()
	calls := map[string]func(f forge.Forge) error{
		"repos": func(f forge.Forge) error {
			_, err := f.Repos().Get(ctx, "owner", "repo")
			return err
		},
		"pull request diff": func(f forge.Forge) error {
			_, err := f.PullRequests().Diff(ctx, "owner", "repo", 1)
			return err
		},
	}

	for _, tt := range tests {
		for name, call := range calls {
			t.Run(tt.name+"/"+name, func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(tt.status)
				}))
				defer srv.Close()
				origAPI := bitbucketAPI
				setBitbucketAPI(srv.URL + "/2.0")
				defer setBitbucketAPI(origAPI)

				if err := call(New("", srv.Client())); !errors.Is(err, tt.want) {
					t.Errorf("error = %v, want %v", err, tt.want)
				}
			})
		}
	}
}

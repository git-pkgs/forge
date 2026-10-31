//go:build !tinygo

package gitlab

import (
	"context"

	forge "github.com/git-pkgs/forge"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func (f *gitLabForge) GetAuthenticatedUser(ctx context.Context) (*forge.User, error) {
	u, _, err := f.client.Users.CurrentUser(gitlab.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	return &forge.User{
		Login:     u.Username,
		Name:      u.Name,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		HTMLURL:   u.WebURL,
	}, nil
}

package gitea

import (
	"context"

	forge "github.com/git-pkgs/forge"
)

func (f *giteaForge) GetAuthenticatedUser(ctx context.Context) (*forge.User, error) {
	u, resp, err := f.client.GetMyUserInfo()
	if err != nil {
		return nil, wrapErr("get authenticated user", resp, err)
	}

	return &forge.User{
		Login:     u.UserName,
		Name:      u.FullName,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		HTMLURL:   u.HTMLURL,
	}, nil
}

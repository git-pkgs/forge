package github

import (
	"context"

	forge "github.com/git-pkgs/forge"
)

func (f *gitHubForge) GetAuthenticatedUser(ctx context.Context) (*forge.User, error) {
	// An empty username asks GitHub for the authenticated user (GET /user).
	u, _, err := f.client.Users.Get(ctx, "")
	if err != nil {
		return nil, err
	}

	return &forge.User{
		Login:     u.GetLogin(),
		Name:      u.GetName(),
		Email:     u.GetEmail(),
		AvatarURL: u.GetAvatarURL(),
		HTMLURL:   u.GetHTMLURL(),
	}, nil
}

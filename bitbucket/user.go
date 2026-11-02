package bitbucket

import (
	"context"

	forge "github.com/git-pkgs/forge"
)

type bbUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Links       struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Avatar struct {
			Href string `json:"href"`
		} `json:"avatar"`
	} `json:"links"`
}

func (f *bitbucketForge) GetAuthenticatedUser(ctx context.Context) (*forge.User, error) {
	rs := &bitbucketRepoService{token: f.token, httpClient: f.httpClient}
	var bb bbUser
	if err := rs.getJSON(ctx, bitbucketAPI+"/user", &bb); err != nil {
		return nil, err
	}

	// Bitbucket only returns email addresses from the separate /user/emails
	// endpoint, so Email is left empty.
	return &forge.User{
		Login:     bb.Username,
		Name:      bb.DisplayName,
		AvatarURL: bb.Links.Avatar.Href,
		HTMLURL:   bb.Links.HTML.Href,
	}, nil
}

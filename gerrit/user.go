package gerrit

import (
	"context"
	"net/http"

	forge "github.com/git-pkgs/forge"
)

func (f *gerritForge) GetAuthenticatedUser(ctx context.Context) (*forge.User, error) {
	var account gerritAccountInfo
	if err := f.doJSON(ctx, http.MethodGet, "/accounts/self", nil, nil, &account); err != nil {
		return nil, err
	}

	user := (&gerritPRService{forge: f}).convertAccount(&account)
	return &user, nil
}

package gitea

import (
	"context"
	"errors"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"strings"

	"code.gitea.io/sdk/gitea"
)

type giteaCommitService struct {
	client *gitea.Client
}

func (f *giteaForge) Commits() forge.CommitService {
	return &giteaCommitService{client: f.client}
}

// ResolveCommit returns the full commit SHA for ref in owner/repo. Gitea and
// Forgejo accept a git ref or a commit SHA on the single-commit endpoint and
// resolve branches, tags and abbreviated SHAs the same way git does.
//
// The context is unused because the Gitea SDK only takes a context on the
// client itself, which is shared across calls, so a per-request deadline
// cannot be applied without racing other callers.
func (s *giteaCommitService) ResolveCommit(_ context.Context, owner, repo, ref string) (string, error) {
	if err := forge.ValidateCommitRef(owner, repo, ref); err != nil {
		return "", err
	}
	if forge.IsFullCommitSHA(ref) {
		return strings.ToLower(ref), nil
	}

	commit, resp, err := s.client.GetSingleCommit(owner, repo, ref)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return "", forge.CommitRefError(owner, repo, ref, forge.ErrNotFound)
		}
		return "", forge.CommitRefError(owner, repo, ref, err)
	}
	if commit == nil || commit.CommitMeta == nil {
		return "", forge.CommitRefError(owner, repo, ref, errors.New("empty response"))
	}
	return forge.ResolvedCommitSHA(owner, repo, ref, commit.SHA)
}

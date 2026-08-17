package gitlab

import (
	"context"
	"errors"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabCommitService struct {
	client *gitlab.Client
}

func (f *gitLabForge) Commits() forge.CommitService {
	return &gitLabCommitService{client: f.client}
}

// ResolveCommit returns the full commit SHA for ref in owner/repo. GitLab's
// single-commit endpoint accepts a branch name, a tag name or a commit SHA,
// and it dereferences annotated tags to the commit they point at.
func (s *gitLabCommitService) ResolveCommit(ctx context.Context, owner, repo, ref string) (string, error) {
	if err := forge.ValidateCommitRef(owner, repo, ref); err != nil {
		return "", err
	}
	if forge.IsFullCommitSHA(ref) {
		return strings.ToLower(ref), nil
	}

	pid := owner + "/" + repo
	commit, resp, err := s.client.Commits.GetCommit(pid, ref, nil, gitlab.WithContext(ctx))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return "", forge.CommitRefError(owner, repo, ref, forge.ErrNotFound)
		}
		return "", forge.CommitRefError(owner, repo, ref, err)
	}
	if commit == nil {
		return "", forge.CommitRefError(owner, repo, ref, errors.New("empty response"))
	}
	return forge.ResolvedCommitSHA(owner, repo, ref, commit.ID)
}

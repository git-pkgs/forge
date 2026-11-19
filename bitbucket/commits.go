package bitbucket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	forge "github.com/git-pkgs/forge"
)

type bitbucketCommitService struct {
	token      string
	httpClient *http.Client
}

func (f *bitbucketForge) Commits() forge.CommitService {
	return &bitbucketCommitService{token: f.token, httpClient: f.httpClient}
}

type bbRef struct {
	Target struct {
		Hash string `json:"hash"`
	} `json:"target"`
}

// ResolveCommit returns the full commit SHA for ref in owner/repo. The
// single-commit endpoint resolves branches, tags and abbreviated SHAs, but it
// takes the ref as a path segment and can't route one that contains a slash,
// such as "feature/login". Those are looked up as a branch and then as a tag.
func (s *bitbucketCommitService) ResolveCommit(ctx context.Context, owner, repo, ref string) (string, error) {
	if err := forge.ValidateCommitRef(owner, repo, ref); err != nil {
		return "", err
	}
	if forge.IsFullCommitSHA(ref) {
		return strings.ToLower(ref), nil
	}

	rs := &bitbucketRepoService{token: s.token, httpClient: s.httpClient}
	var sha string
	if strings.Contains(ref, "/") {
		hash, err := resolveSlashedRef(ctx, rs, owner, repo, ref)
		if err != nil {
			return "", forge.CommitRefError(owner, repo, ref, err)
		}
		sha = hash
	} else {
		u := fmt.Sprintf("%s/repositories/%s/%s/commit/%s", bitbucketAPI, owner, repo, url.PathEscape(ref))
		var commit struct {
			Hash string `json:"hash"`
		}
		if err := rs.getJSON(ctx, u, &commit); err != nil {
			return "", forge.CommitRefError(owner, repo, ref, err)
		}
		sha = commit.Hash
	}
	return forge.ResolvedCommitSHA(owner, repo, ref, sha)
}

// resolveSlashedRef returns the commit hash of the branch or tag named ref.
// Bitbucket only accepts a ref containing a slash on its refs endpoints, where
// it is escaped into one path segment. Branches are tried first because that
// is where slashed names are common. It returns forge.ErrNotFound when
// neither exists.
func resolveSlashedRef(ctx context.Context, rs *bitbucketRepoService, owner, repo, ref string) (string, error) {
	for _, kind := range []string{"branches", "tags"} {
		u := fmt.Sprintf("%s/repositories/%s/%s/refs/%s/%s", bitbucketAPI, owner, repo, kind, url.PathEscape(ref))
		var r bbRef
		err := rs.getJSON(ctx, u, &r)
		if errors.Is(err, forge.ErrNotFound) {
			continue
		}
		if err != nil {
			return "", err
		}
		return r.Target.Hash, nil
	}
	return "", forge.ErrNotFound
}

package gitea

import (
	"context"
	"net/http"

	forge "github.com/git-pkgs/forge"
	"code.gitea.io/sdk/gitea"
)

func convertGiteaReaction(r *gitea.Reaction) forge.Reaction {
	result := forge.Reaction{
		Content: r.Reaction,
	}
	if r.User != nil {
		result.User = r.User.UserName
	}
	return result
}

func (s *giteaIssueService) ListReactions(ctx context.Context, owner, repo string, number int, commentID int64) ([]forge.Reaction, error) {
	reactions, resp, err := s.client.GetIssueCommentReactions(owner, repo, commentID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	var all []forge.Reaction
	for _, r := range reactions {
		all = append(all, convertGiteaReaction(r))
	}
	return all, nil
}

func (s *giteaIssueService) AddReaction(ctx context.Context, owner, repo string, number int, commentID int64, reaction string) (*forge.Reaction, error) {
	r, resp, err := s.client.PostIssueCommentReaction(owner, repo, commentID, reaction)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaReaction(r)
	return &result, nil
}

func (s *giteaPRService) ListReactions(ctx context.Context, owner, repo string, number int, commentID int64) ([]forge.Reaction, error) {
	// Gitea uses the same issue comment reactions API for PR comments
	reactions, resp, err := s.client.GetIssueCommentReactions(owner, repo, commentID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	var all []forge.Reaction
	for _, r := range reactions {
		all = append(all, convertGiteaReaction(r))
	}
	return all, nil
}

func (s *giteaPRService) AddReaction(ctx context.Context, owner, repo string, number int, commentID int64, reaction string) (*forge.Reaction, error) {
	r, resp, err := s.client.PostIssueCommentReaction(owner, repo, commentID, reaction)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaReaction(r)
	return &result, nil
}

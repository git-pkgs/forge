package github

import (
	"context"
	"net/http"

	forge "github.com/git-pkgs/forge"
	"github.com/google/go-github/v82/github"
)

func convertGitHubReaction(r *github.Reaction) forge.Reaction {
	result := forge.Reaction{
		ID:      r.GetID(),
		Content: r.GetContent(),
	}
	if u := r.GetUser(); u != nil {
		result.User = u.GetLogin()
	}
	return result
}

func (s *gitHubIssueService) ListReactions(ctx context.Context, owner, repo string, number int, commentID int64) ([]forge.Reaction, error) {
	var all []forge.Reaction
	opts := &github.ListReactionOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		reactions, resp, err := s.client.Reactions.ListIssueCommentReactions(ctx, owner, repo, commentID, opts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, r := range reactions {
			all = append(all, convertGitHubReaction(r))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

func (s *gitHubIssueService) AddReaction(ctx context.Context, owner, repo string, number int, commentID int64, reaction string) (*forge.Reaction, error) {
	r, resp, err := s.client.Reactions.CreateIssueCommentReaction(ctx, owner, repo, commentID, reaction)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubReaction(r)
	return &result, nil
}

func (s *gitHubPRService) ListReactions(ctx context.Context, owner, repo string, number int, commentID int64) ([]forge.Reaction, error) {
	// GitHub uses the same issue comment reactions API for PR comments
	var all []forge.Reaction
	opts := &github.ListReactionOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		reactions, resp, err := s.client.Reactions.ListIssueCommentReactions(ctx, owner, repo, commentID, opts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, r := range reactions {
			all = append(all, convertGitHubReaction(r))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

func (s *gitHubPRService) AddReaction(ctx context.Context, owner, repo string, number int, commentID int64, reaction string) (*forge.Reaction, error) {
	r, resp, err := s.client.Reactions.CreateIssueCommentReaction(ctx, owner, repo, commentID, reaction)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubReaction(r)
	return &result, nil
}

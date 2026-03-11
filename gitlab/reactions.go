package gitlab

import (
	"context"
	"net/http"

	forge "github.com/git-pkgs/forge"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func convertGitLabAwardEmoji(e *gitlab.AwardEmoji) forge.Reaction {
	return forge.Reaction{
		ID:      e.ID,
		User:    e.User.Username,
		Content: e.Name,
	}
}

func (s *gitLabIssueService) ListReactions(ctx context.Context, owner, repo string, number int, commentID int64) ([]forge.Reaction, error) {
	pid := owner + "/" + repo
	emojis, resp, err := s.client.AwardEmoji.ListIssuesAwardEmojiOnNote(pid, int64(number), commentID, &gitlab.ListAwardEmojiOptions{})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	var all []forge.Reaction
	for _, e := range emojis {
		all = append(all, convertGitLabAwardEmoji(e))
	}
	return all, nil
}

func (s *gitLabIssueService) AddReaction(ctx context.Context, owner, repo string, number int, commentID int64, reaction string) (*forge.Reaction, error) {
	pid := owner + "/" + repo
	emoji, resp, err := s.client.AwardEmoji.CreateIssuesAwardEmojiOnNote(pid, int64(number), commentID, &gitlab.CreateAwardEmojiOptions{
		Name: reaction,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabAwardEmoji(emoji)
	return &result, nil
}

func (s *gitLabPRService) ListReactions(ctx context.Context, owner, repo string, number int, commentID int64) ([]forge.Reaction, error) {
	pid := owner + "/" + repo
	emojis, resp, err := s.client.AwardEmoji.ListMergeRequestAwardEmojiOnNote(pid, int64(number), commentID, &gitlab.ListAwardEmojiOptions{})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	var all []forge.Reaction
	for _, e := range emojis {
		all = append(all, convertGitLabAwardEmoji(e))
	}
	return all, nil
}

func (s *gitLabPRService) AddReaction(ctx context.Context, owner, repo string, number int, commentID int64, reaction string) (*forge.Reaction, error) {
	pid := owner + "/" + repo
	emoji, resp, err := s.client.AwardEmoji.CreateMergeRequestAwardEmojiOnNote(pid, int64(number), commentID, &gitlab.CreateAwardEmojiOptions{
		Name: reaction,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabAwardEmoji(emoji)
	return &result, nil
}

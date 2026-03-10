package gitlab

import (
	"context"
	"fmt"
	forge "github.com/git-pkgs/forge"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabReviewService struct {
	client *gitlab.Client
}

func (f *gitLabForge) Reviews() forge.ReviewService {
	return &gitLabReviewService{client: f.client}
}

func (s *gitLabReviewService) List(ctx context.Context, owner, repo string, number int, opts forge.ListReviewOpts) ([]forge.Review, error) {
	pid := owner + "/" + repo

	approvals, resp, err := s.client.MergeRequestApprovals.GetConfiguration(pid, int64(number))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	var reviews []forge.Review
	for _, a := range approvals.ApprovedBy {
		if a.User == nil {
			continue
		}
		reviews = append(reviews, forge.Review{
			State: forge.ReviewApproved,
			Author: forge.User{
				Login:     a.User.Username,
				Name:      a.User.Name,
				AvatarURL: a.User.AvatarURL,
				HTMLURL:   a.User.WebURL,
			},
		})
	}

	return reviews, nil
}

func (s *gitLabReviewService) Submit(ctx context.Context, owner, repo string, number int, opts forge.SubmitReviewOpts) (*forge.Review, error) {
	pid := owner + "/" + repo

	switch opts.State {
	case forge.ReviewApproved:
		_, resp, err := s.client.MergeRequestApprovals.ApproveMergeRequest(pid, int64(number), nil)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		result := &forge.Review{State: forge.ReviewApproved}
		return result, nil

	case forge.ReviewChangesRequested:
		return nil, fmt.Errorf("requesting changes: %w", forge.ErrNotSupported)

	default:
		// For comment-only reviews, add a note to the MR
		n, resp, err := s.client.Notes.CreateMergeRequestNote(pid, int64(number), &gitlab.CreateMergeRequestNoteOptions{
			Body: gitlab.Ptr(opts.Body),
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		result := &forge.Review{
			ID:    int64(n.ID),
			State: forge.ReviewCommented,
			Body:  n.Body,
			Author: forge.User{
				Login:     n.Author.Username,
				Name:      n.Author.Name,
				AvatarURL: n.Author.AvatarURL,
				HTMLURL:   n.Author.WebURL,
			},
		}
		if n.CreatedAt != nil {
			result.SubmittedAt = *n.CreatedAt
		}
		return result, nil
	}
}

func (s *gitLabReviewService) RequestReviewers(ctx context.Context, owner, repo string, number int, users []string) error {
	pid := owner + "/" + repo

	// GitLab requires user IDs, not usernames. Resolve them.
	ids, err := s.resolveUserIDs(users)
	if err != nil {
		return err
	}

	_, resp, err := s.client.MergeRequests.UpdateMergeRequest(pid, int64(number), &gitlab.UpdateMergeRequestOptions{
		ReviewerIDs: &ids,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabReviewService) RemoveReviewers(ctx context.Context, owner, repo string, number int, users []string) error {
	pid := owner + "/" + repo

	// Get current reviewers
	mr, resp, err := s.client.MergeRequests.GetMergeRequest(pid, int64(number), nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}

	removeSet := make(map[string]bool)
	for _, u := range users {
		removeSet[u] = true
	}

	var remaining []int64
	for _, r := range mr.Reviewers {
		if !removeSet[r.Username] {
			remaining = append(remaining, int64(r.ID))
		}
	}

	_, resp, err = s.client.MergeRequests.UpdateMergeRequest(pid, int64(number), &gitlab.UpdateMergeRequestOptions{
		ReviewerIDs: &remaining,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabReviewService) resolveUserIDs(usernames []string) ([]int64, error) {
	ids := make([]int64, 0, len(usernames))
	for _, username := range usernames {
		users, _, err := s.client.Users.ListUsers(&gitlab.ListUsersOptions{
			Username: gitlab.Ptr(username),
		})
		if err != nil {
			return nil, fmt.Errorf("looking up user %q: %w", username, err)
		}
		if len(users) == 0 {
			return nil, fmt.Errorf("user %q not found", username)
		}
		ids = append(ids, int64(users[0].ID))
	}
	return ids, nil
}

package gitea

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"strings"

	"code.gitea.io/sdk/gitea"
)

type giteaReviewService struct {
	client *gitea.Client
}

func (f *giteaForge) Reviews() forge.ReviewService {
	return &giteaReviewService{client: f.client}
}

func convertGiteaReviewState(s gitea.ReviewStateType) forge.ReviewState {
	switch s {
	case gitea.ReviewStateApproved:
		return forge.ReviewApproved
	case gitea.ReviewStateRequestChanges:
		return forge.ReviewChangesRequested
	case gitea.ReviewStateComment:
		return forge.ReviewCommented
	case gitea.ReviewStateRequestReview:
		return forge.ReviewPending
	default:
		return forge.ReviewState(strings.ToLower(string(s)))
	}
}

func convertGiteaReview(r *gitea.PullReview) forge.Review {
	result := forge.Review{
		ID:    r.ID,
		State: convertGiteaReviewState(r.State),
		Body:  r.Body,
	}

	if r.Reviewer != nil {
		result.Author = forge.User{
			Login:     r.Reviewer.UserName,
			AvatarURL: r.Reviewer.AvatarURL,
		}
	}

	if r.HTMLURL != "" {
		result.HTMLURL = r.HTMLURL
	}

	if !r.Submitted.IsZero() {
		result.SubmittedAt = r.Submitted
	}

	return result
}

func (s *giteaReviewService) List(ctx context.Context, owner, repo string, number int, opts forge.ListReviewOpts) ([]forge.Review, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	var all []forge.Review
	for {
		reviews, resp, err := s.client.ListPullReviews(owner, repo, int64(number), gitea.ListPullReviewsOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, r := range reviews {
			all = append(all, convertGiteaReview(r))
		}
		if len(reviews) < perPage || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		page++
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func forgeStateToGiteaType(state forge.ReviewState) gitea.ReviewStateType {
	switch state {
	case forge.ReviewApproved:
		return gitea.ReviewStateApproved
	case forge.ReviewChangesRequested:
		return gitea.ReviewStateRequestChanges
	default:
		return gitea.ReviewStateComment
	}
}

func (s *giteaReviewService) Submit(ctx context.Context, owner, repo string, number int, opts forge.SubmitReviewOpts) (*forge.Review, error) {
	review, resp, err := s.client.CreatePullReview(owner, repo, int64(number), gitea.CreatePullReviewOptions{
		State: forgeStateToGiteaType(opts.State),
		Body:  opts.Body,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaReview(review)
	return &result, nil
}

func (s *giteaReviewService) RequestReviewers(ctx context.Context, owner, repo string, number int, users []string) error {
	resp, err := s.client.CreateReviewRequests(owner, repo, int64(number), gitea.PullReviewRequestOptions{
		Reviewers: users,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *giteaReviewService) RemoveReviewers(ctx context.Context, owner, repo string, number int, users []string) error {
	resp, err := s.client.DeleteReviewRequests(owner, repo, int64(number), gitea.PullReviewRequestOptions{
		Reviewers: users,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

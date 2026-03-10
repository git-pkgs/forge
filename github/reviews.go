package github

import (
	"context"
	"net/http"
	"strings"

	forge "github.com/git-pkgs/forge"

	"github.com/google/go-github/v82/github"
)

type gitHubReviewService struct {
	client *github.Client
}

func (f *gitHubForge) Reviews() forge.ReviewService {
	return &gitHubReviewService{client: f.client}
}

func convertGitHubReviewState(s string) forge.ReviewState {
	switch strings.ToUpper(s) {
	case "APPROVED":
		return forge.ReviewApproved
	case "CHANGES_REQUESTED":
		return forge.ReviewChangesRequested
	case "COMMENTED":
		return forge.ReviewCommented
	case "DISMISSED":
		return forge.ReviewDismissed
	case "PENDING":
		return forge.ReviewPending
	default:
		return forge.ReviewState(strings.ToLower(s))
	}
}

func convertGitHubReview(r *github.PullRequestReview) forge.Review {
	result := forge.Review{
		ID:      r.GetID(),
		State:   convertGitHubReviewState(r.GetState()),
		Body:    r.GetBody(),
		HTMLURL: r.GetHTMLURL(),
	}

	if u := r.GetUser(); u != nil {
		result.Author = forge.User{
			Login:     u.GetLogin(),
			AvatarURL: u.GetAvatarURL(),
			HTMLURL:   u.GetHTMLURL(),
		}
	}

	if t := r.GetSubmittedAt(); !t.IsZero() {
		result.SubmittedAt = t.Time
	}

	return result
}

func (s *gitHubReviewService) List(ctx context.Context, owner, repo string, number int, opts forge.ListReviewOpts) ([]forge.Review, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.ListOptions{PerPage: perPage, Page: page}

	var all []forge.Review
	for {
		reviews, resp, err := s.client.PullRequests.ListReviews(ctx, owner, repo, number, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, r := range reviews {
			all = append(all, convertGitHubReview(r))
		}
		if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		ghOpts.Page = resp.NextPage
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func forgeStateToGitHubEvent(state forge.ReviewState) string {
	switch state {
	case forge.ReviewApproved:
		return "APPROVE"
	case forge.ReviewChangesRequested:
		return "REQUEST_CHANGES"
	default:
		return "COMMENT"
	}
}

func (s *gitHubReviewService) Submit(ctx context.Context, owner, repo string, number int, opts forge.SubmitReviewOpts) (*forge.Review, error) {
	event := forgeStateToGitHubEvent(opts.State)
	req := &github.PullRequestReviewRequest{
		Body:  &opts.Body,
		Event: &event,
	}

	review, resp, err := s.client.PullRequests.CreateReview(ctx, owner, repo, number, req)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubReview(review)
	return &result, nil
}

func (s *gitHubReviewService) RequestReviewers(ctx context.Context, owner, repo string, number int, users []string) error {
	_, resp, err := s.client.PullRequests.RequestReviewers(ctx, owner, repo, number, github.ReviewersRequest{
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

func (s *gitHubReviewService) RemoveReviewers(ctx context.Context, owner, repo string, number int, users []string) error {
	resp, err := s.client.PullRequests.RemoveReviewers(ctx, owner, repo, number, github.ReviewersRequest{
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

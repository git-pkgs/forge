package bitbucket

import (
	"context"
	"fmt"
	forge "github.com/git-pkgs/forge"
	"net/http"
)

type bitbucketReviewService struct {
	token      string
	httpClient *http.Client
}

func (f *bitbucketForge) Reviews() forge.ReviewService {
	return &bitbucketReviewService{token: f.token, httpClient: f.httpClient}
}

func (s *bitbucketReviewService) doJSON(ctx context.Context, method, url string, body any, v any) error {
	rs := &bitbucketRepoService{token: s.token, httpClient: s.httpClient}
	return rs.doJSON(ctx, method, url, body, v)
}

type bbParticipant struct {
	User struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	} `json:"user"`
	Role     string `json:"role"`
	Approved bool   `json:"approved"`
}

type bbPRDetail struct {
	Participants []bbParticipant `json:"participants"`
}

func (s *bitbucketReviewService) List(ctx context.Context, owner, repo string, number int, opts forge.ListReviewOpts) ([]forge.Review, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d", bitbucketAPI, owner, repo, number)
	var bb bbPRDetail
	if err := s.doJSON(ctx, http.MethodGet, url, nil, &bb); err != nil {
		return nil, err
	}

	var reviews []forge.Review
	for _, p := range bb.Participants {
		if p.Role != "REVIEWER" {
			continue
		}
		state := forge.ReviewCommented
		if p.Approved {
			state = forge.ReviewApproved
		}
		reviews = append(reviews, forge.Review{
			State: state,
			Author: forge.User{
				Login: p.User.Username,
			},
		})
	}

	return reviews, nil
}

func (s *bitbucketReviewService) Submit(ctx context.Context, owner, repo string, number int, opts forge.SubmitReviewOpts) (*forge.Review, error) {
	switch opts.State {
	case forge.ReviewApproved:
		url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/approve", bitbucketAPI, owner, repo, number)
		if err := s.doJSON(ctx, http.MethodPost, url, nil, nil); err != nil {
			return nil, err
		}
		return &forge.Review{State: forge.ReviewApproved}, nil

	case forge.ReviewChangesRequested:
		return nil, fmt.Errorf("requesting changes: %w", forge.ErrNotSupported)

	default:
		// Post a comment as the review
		reqBody := map[string]any{
			"content": map[string]string{"raw": opts.Body},
		}
		url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments", bitbucketAPI, owner, repo, number)
		if err := s.doJSON(ctx, http.MethodPost, url, reqBody, nil); err != nil {
			return nil, err
		}
		return &forge.Review{State: forge.ReviewCommented, Body: opts.Body}, nil
	}
}

func (s *bitbucketReviewService) RequestReviewers(ctx context.Context, owner, repo string, number int, users []string) error {
	// Bitbucket sets reviewers on the PR body. Get current PR, add reviewers, update.
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d", bitbucketAPI, owner, repo, number)
	var bb bbPullRequest
	if err := s.doJSON(ctx, http.MethodGet, url, nil, &bb); err != nil {
		return err
	}

	existing := make(map[string]bool)
	var reviewers []map[string]string
	for _, r := range bb.Reviewers {
		existing[r.Username] = true
		reviewers = append(reviewers, map[string]string{"username": r.Username})
	}
	for _, u := range users {
		if !existing[u] {
			reviewers = append(reviewers, map[string]string{"username": u})
		}
	}

	body := map[string]any{"reviewers": reviewers}
	return s.doJSON(ctx, http.MethodPut, url, body, nil)
}

func (s *bitbucketReviewService) RemoveReviewers(ctx context.Context, owner, repo string, number int, users []string) error {
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d", bitbucketAPI, owner, repo, number)
	var bb bbPullRequest
	if err := s.doJSON(ctx, http.MethodGet, url, nil, &bb); err != nil {
		return err
	}

	removeSet := make(map[string]bool)
	for _, u := range users {
		removeSet[u] = true
	}

	var reviewers []map[string]string
	for _, r := range bb.Reviewers {
		if !removeSet[r.Username] {
			reviewers = append(reviewers, map[string]string{"username": r.Username})
		}
	}

	body := map[string]any{"reviewers": reviewers}
	return s.doJSON(ctx, http.MethodPut, url, body, nil)
}

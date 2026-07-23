package gerrit

import (
	"context"
	"net/http"
	"strconv"
	"time"

	forge "github.com/git-pkgs/forge"
)

type gerritReviewService struct {
	forge *gerritForge
}

func (f *gerritForge) Reviews() forge.ReviewService {
	return &gerritReviewService{forge: f}
}

func (s *gerritReviewService) List(_ context.Context, _, _ string, _ int, _ forge.ListReviewOpts) ([]forge.Review, error) {
	return nil, forge.ErrNotSupported
}

func (s *gerritReviewService) Submit(ctx context.Context, owner, repo string, number int, opts forge.SubmitReviewOpts) (*forge.Review, error) {
	state := opts.State
	if state == "" {
		state = forge.ReviewCommented
	}

	body := map[string]any{}
	if opts.Body != "" {
		body["message"] = opts.Body
	}
	labels := map[string]int{}
	switch state {
	case forge.ReviewApproved:
		labels["Code-Review"] = 2
	case forge.ReviewChangesRequested:
		labels["Code-Review"] = -2
	case forge.ReviewCommented:
	default:
		return nil, forge.ErrNotSupported
	}
	if len(labels) > 0 {
		body["labels"] = labels
	}

	if err := s.forge.doJSON(ctx, http.MethodPost, "/changes/"+encodeID(strconv.Itoa(number))+"/revisions/current/review", nil, body, nil); err != nil {
		return nil, err
	}
	return &forge.Review{
		State:       state,
		Body:        opts.Body,
		HTMLURL:     (&gerritPRService{forge: s.forge}).changeURL(projectName(owner, repo), number),
		SubmittedAt: time.Now().UTC(),
	}, nil
}

func (s *gerritReviewService) RequestReviewers(ctx context.Context, owner, repo string, number int, users []string) error {
	for _, user := range users {
		body := map[string]string{"reviewer": user}
		if err := s.forge.doJSON(ctx, http.MethodPost, "/changes/"+encodeID(strconv.Itoa(number))+"/reviewers", nil, body, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *gerritReviewService) RemoveReviewers(ctx context.Context, owner, repo string, number int, users []string) error {
	for _, user := range users {
		if err := s.forge.doJSON(ctx, http.MethodDelete, "/changes/"+encodeID(strconv.Itoa(number))+"/reviewers/"+encodeID(user), nil, nil, nil); err != nil {
			return err
		}
	}
	return nil
}

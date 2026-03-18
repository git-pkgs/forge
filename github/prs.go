package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	forge "github.com/git-pkgs/forge"

	"github.com/google/go-github/v82/github"
)

type gitHubPRService struct {
	client *github.Client
}

func (f *gitHubForge) PullRequests() forge.PullRequestService {
	return &gitHubPRService{client: f.client}
}

func convertGitHubPR(pr *github.PullRequest) forge.PullRequest {
	result := forge.PullRequest{
		Number:       pr.GetNumber(),
		Title:        pr.GetTitle(),
		Body:         pr.GetBody(),
		State:        pr.GetState(),
		Draft:        pr.GetDraft(),
		Mergeable:    pr.GetMergeable(),
		Merged:       pr.GetMerged(),
		Comments:     pr.GetComments(),
		Additions:    pr.GetAdditions(),
		Deletions:    pr.GetDeletions(),
		ChangedFiles: pr.GetChangedFiles(),
		HTMLURL:      pr.GetHTMLURL(),
		DiffURL:      pr.GetDiffURL(),
	}

	if pr.GetMerged() {
		result.State = "merged"
	}

	if u := pr.GetUser(); u != nil {
		result.Author = forge.User{
			Login:     u.GetLogin(),
			AvatarURL: u.GetAvatarURL(),
			HTMLURL:   u.GetHTMLURL(),
		}
	}

	for _, a := range pr.Assignees {
		result.Assignees = append(result.Assignees, forge.User{
			Login:     a.GetLogin(),
			AvatarURL: a.GetAvatarURL(),
			HTMLURL:   a.GetHTMLURL(),
		})
	}

	for _, r := range pr.RequestedReviewers {
		result.Reviewers = append(result.Reviewers, forge.User{
			Login:     r.GetLogin(),
			AvatarURL: r.GetAvatarURL(),
			HTMLURL:   r.GetHTMLURL(),
		})
	}

	for _, l := range pr.Labels {
		result.Labels = append(result.Labels, forge.Label{
			Name:        l.GetName(),
			Color:       l.GetColor(),
			Description: l.GetDescription(),
		})
	}

	if m := pr.GetMilestone(); m != nil {
		result.Milestone = &forge.Milestone{
			Title:       m.GetTitle(),
			Number:      m.GetNumber(),
			Description: m.GetDescription(),
			State:       m.GetState(),
		}
	}

	if h := pr.GetHead(); h != nil {
		result.Head = h.GetRef()
	}
	if b := pr.GetBase(); b != nil {
		result.Base = b.GetRef()
	}

	if u := pr.GetMergedBy(); u != nil {
		result.MergedBy = &forge.User{
			Login:     u.GetLogin(),
			AvatarURL: u.GetAvatarURL(),
			HTMLURL:   u.GetHTMLURL(),
		}
	}

	if t := pr.GetCreatedAt(); !t.IsZero() {
		result.CreatedAt = t.Time
	}
	if t := pr.GetUpdatedAt(); !t.IsZero() {
		result.UpdatedAt = t.Time
	}
	if t := pr.GetMergedAt(); !t.IsZero() {
		mt := t.Time
		result.MergedAt = &mt
	}
	if t := pr.GetClosedAt(); !t.IsZero() {
		ct := t.Time
		result.ClosedAt = &ct
	}

	return result
}

func (s *gitHubPRService) Get(ctx context.Context, owner, repo string, number int) (*forge.PullRequest, error) {
	pr, resp, err := s.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubPR(pr)
	return &result, nil
}

func (s *gitHubPRService) List(ctx context.Context, owner, repo string, opts forge.ListPROpts) ([]forge.PullRequest, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.PullRequestListOptions{
		State:       opts.State,
		Head:        opts.Head,
		Base:        opts.Base,
		Sort:        opts.Sort,
		Direction:   opts.Order,
		ListOptions: github.ListOptions{PerPage: perPage, Page: page},
	}
	if ghOpts.State == "" {
		ghOpts.State = stateOpen
	}

	var all []forge.PullRequest
	for {
		prs, resp, err := s.client.PullRequests.List(ctx, owner, repo, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, pr := range prs {
			all = append(all, convertGitHubPR(pr))
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

func (s *gitHubPRService) Create(ctx context.Context, owner, repo string, opts forge.CreatePROpts) (*forge.PullRequest, error) {
	req := &github.NewPullRequest{
		Title: &opts.Title,
		Head:  &opts.Head,
		Base:  &opts.Base,
		Draft: &opts.Draft,
	}
	if opts.Body != "" {
		req.Body = &opts.Body
	}

	pr, resp, err := s.client.PullRequests.Create(ctx, owner, repo, req)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	// Add reviewers if requested
	if len(opts.Reviewers) > 0 {
		if _, _, err := s.client.PullRequests.RequestReviewers(ctx, owner, repo, pr.GetNumber(), github.ReviewersRequest{
			Reviewers: opts.Reviewers,
		}); err != nil {
			return nil, fmt.Errorf("PR created but failed to add reviewers: %w", err)
		}
	}

	// Add assignees, labels, and milestone in a single edit if any are set.
	issueReq := &github.IssueRequest{}
	needsEdit := false
	if len(opts.Assignees) > 0 {
		issueReq.Assignees = &opts.Assignees
		needsEdit = true
	}
	if len(opts.Labels) > 0 {
		issueReq.Labels = &opts.Labels
		needsEdit = true
	}
	if opts.Milestone != "" {
		n, err := strconv.Atoi(opts.Milestone)
		if err == nil {
			issueReq.Milestone = &n
			needsEdit = true
		}
	}
	if needsEdit {
		if _, _, err := s.client.Issues.Edit(ctx, owner, repo, pr.GetNumber(), issueReq); err != nil {
			return nil, fmt.Errorf("PR created but failed to set assignees/labels/milestone: %w", err)
		}
	}

	result := convertGitHubPR(pr)
	return &result, nil
}

func (s *gitHubPRService) Update(ctx context.Context, owner, repo string, number int, opts forge.UpdatePROpts) (*forge.PullRequest, error) {
	ghPR := &github.PullRequest{}
	changed := false

	if opts.Title != nil {
		ghPR.Title = opts.Title
		changed = true
	}
	if opts.Body != nil {
		ghPR.Body = opts.Body
		changed = true
	}
	if opts.Base != nil {
		ghPR.Base = &github.PullRequestBranch{Ref: opts.Base}
		changed = true
	}

	if changed {
		pr, resp, err := s.client.PullRequests.Edit(ctx, owner, repo, number, ghPR)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		result := convertGitHubPR(pr)
		return &result, nil
	}

	return s.Get(ctx, owner, repo, number)
}

func (s *gitHubPRService) Close(ctx context.Context, owner, repo string, number int) error {
	state := stateClosed
	_, resp, err := s.client.PullRequests.Edit(ctx, owner, repo, number, &github.PullRequest{
		State: &state,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitHubPRService) Reopen(ctx context.Context, owner, repo string, number int) error {
	state := stateOpen
	_, resp, err := s.client.PullRequests.Edit(ctx, owner, repo, number, &github.PullRequest{
		State: &state,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitHubPRService) Merge(ctx context.Context, owner, repo string, number int, opts forge.MergePROpts) error {
	ghOpts := &github.PullRequestOptions{}
	if opts.Method != "" {
		ghOpts.MergeMethod = opts.Method
	}
	if opts.Title != "" {
		ghOpts.CommitTitle = opts.Title
	}

	// Fetch the PR first if we need to delete the branch, since we need the head ref name.
	var headRef string
	if opts.Delete {
		pr, _, err := s.client.PullRequests.Get(ctx, owner, repo, number)
		if err == nil && pr.GetHead() != nil {
			headRef = pr.GetHead().GetRef()
		}
	}

	_, resp, err := s.client.PullRequests.Merge(ctx, owner, repo, number, opts.Message, ghOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}

	if opts.Delete && headRef != "" {
		if _, err := s.client.Git.DeleteRef(ctx, owner, repo, "heads/"+headRef); err != nil {
			return fmt.Errorf("merged successfully but failed to delete branch %q: %w", headRef, err)
		}
	}

	return nil
}

func (s *gitHubPRService) Diff(ctx context.Context, owner, repo string, number int) (string, error) {
	raw, resp, err := s.client.PullRequests.GetRaw(ctx, owner, repo, number, github.RawOptions{Type: github.Diff})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return "", forge.ErrNotFound
		}
		return "", err
	}
	return raw, nil
}

func (s *gitHubPRService) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*forge.Comment, error) {
	c, resp, err := s.client.Issues.CreateComment(ctx, owner, repo, number, &github.IssueComment{
		Body: &body,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubComment(c)
	return &result, nil
}

func (s *gitHubPRService) ListComments(ctx context.Context, owner, repo string, number int) ([]forge.Comment, error) {
	var all []forge.Comment
	ghOpts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: defaultPageSize},
	}
	for {
		comments, resp, err := s.client.Issues.ListComments(ctx, owner, repo, number, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, c := range comments {
			all = append(all, convertGitHubComment(c))
		}
		if resp.NextPage == 0 {
			break
		}
		ghOpts.Page = resp.NextPage
	}
	return all, nil
}

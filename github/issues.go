package github

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"strconv"

	"github.com/google/go-github/v82/github"
)

type gitHubIssueService struct {
	client *github.Client
}

func (f *gitHubForge) Issues() forge.IssueService {
	return &gitHubIssueService{client: f.client}
}

func convertGitHubIssue(i *github.Issue) forge.Issue {
	result := forge.Issue{
		Number:  i.GetNumber(),
		Title:   i.GetTitle(),
		Body:    i.GetBody(),
		State:   i.GetState(),
		Locked:  i.GetLocked(),
		HTMLURL: i.GetHTMLURL(),
	}

	if u := i.GetUser(); u != nil {
		result.Author = forge.User{
			Login:     u.GetLogin(),
			AvatarURL: u.GetAvatarURL(),
			HTMLURL:   u.GetHTMLURL(),
		}
	}

	for _, a := range i.Assignees {
		result.Assignees = append(result.Assignees, forge.User{
			Login:     a.GetLogin(),
			AvatarURL: a.GetAvatarURL(),
			HTMLURL:   a.GetHTMLURL(),
		})
	}

	for _, l := range i.Labels {
		result.Labels = append(result.Labels, forge.Label{
			Name:        l.GetName(),
			Color:       l.GetColor(),
			Description: l.GetDescription(),
		})
	}

	if m := i.GetMilestone(); m != nil {
		result.Milestone = &forge.Milestone{
			Title:       m.GetTitle(),
			Number:      m.GetNumber(),
			Description: m.GetDescription(),
			State:       m.GetState(),
		}
		if m.DueOn != nil {
			t := m.DueOn.Time
			result.Milestone.DueDate = &t
		}
	}

	result.Comments = i.GetComments()

	if t := i.GetCreatedAt(); !t.IsZero() {
		result.CreatedAt = t.Time
	}
	if t := i.GetUpdatedAt(); !t.IsZero() {
		result.UpdatedAt = t.Time
	}
	if t := i.GetClosedAt(); !t.IsZero() {
		ct := t.Time
		result.ClosedAt = &ct
	}

	return result
}

func convertGitHubComment(c *github.IssueComment) forge.Comment {
	result := forge.Comment{
		ID:      c.GetID(),
		Body:    c.GetBody(),
		HTMLURL: c.GetHTMLURL(),
	}
	if u := c.GetUser(); u != nil {
		result.Author = forge.User{
			Login:     u.GetLogin(),
			AvatarURL: u.GetAvatarURL(),
			HTMLURL:   u.GetHTMLURL(),
		}
	}
	if t := c.GetCreatedAt(); !t.IsZero() {
		result.CreatedAt = t.Time
	}
	if t := c.GetUpdatedAt(); !t.IsZero() {
		result.UpdatedAt = t.Time
	}
	return result
}

func (s *gitHubIssueService) Get(ctx context.Context, owner, repo string, number int) (*forge.Issue, error) {
	i, resp, err := s.client.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubIssue(i)
	return &result, nil
}

func (s *gitHubIssueService) List(ctx context.Context, owner, repo string, opts forge.ListIssueOpts) ([]forge.Issue, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.IssueListByRepoOptions{
		State:       opts.State,
		Assignee:    opts.Assignee,
		Sort:        opts.Sort,
		Direction:   opts.Order,
		ListOptions: github.ListOptions{PerPage: perPage, Page: page},
	}

	if len(opts.Labels) > 0 {
		ghOpts.Labels = opts.Labels
	}

	var all []forge.Issue
	for {
		issues, resp, err := s.client.Issues.ListByRepo(ctx, owner, repo, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, i := range issues {
			// GitHub's issue list includes pull requests; skip them
			if i.PullRequestLinks != nil {
				continue
			}
			all = append(all, convertGitHubIssue(i))
		}
		if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		ghOpts.ListOptions.Page = resp.NextPage
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *gitHubIssueService) Create(ctx context.Context, owner, repo string, opts forge.CreateIssueOpts) (*forge.Issue, error) {
	req := &github.IssueRequest{
		Title: &opts.Title,
	}
	if opts.Body != "" {
		req.Body = &opts.Body
	}
	if len(opts.Assignees) > 0 {
		req.Assignees = &opts.Assignees
	}
	if len(opts.Labels) > 0 {
		req.Labels = &opts.Labels
	}
	if opts.Milestone != "" {
		n, err := strconv.Atoi(opts.Milestone)
		if err == nil {
			req.Milestone = &n
		}
	}

	i, resp, err := s.client.Issues.Create(ctx, owner, repo, req)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubIssue(i)
	return &result, nil
}

func (s *gitHubIssueService) Update(ctx context.Context, owner, repo string, number int, opts forge.UpdateIssueOpts) (*forge.Issue, error) {
	req := &github.IssueRequest{}
	changed := false

	if opts.Title != nil {
		req.Title = opts.Title
		changed = true
	}
	if opts.Body != nil {
		req.Body = opts.Body
		changed = true
	}
	if opts.Assignees != nil {
		req.Assignees = &opts.Assignees
		changed = true
	}
	if opts.Labels != nil {
		req.Labels = &opts.Labels
		changed = true
	}
	if opts.Milestone != nil {
		n, err := strconv.Atoi(*opts.Milestone)
		if err == nil {
			req.Milestone = &n
		}
		changed = true
	}

	if !changed {
		return s.Get(ctx, owner, repo, number)
	}

	i, resp, err := s.client.Issues.Edit(ctx, owner, repo, number, req)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubIssue(i)
	return &result, nil
}

func (s *gitHubIssueService) Close(ctx context.Context, owner, repo string, number int) error {
	state := "closed"
	req := &github.IssueRequest{State: &state}
	_, resp, err := s.client.Issues.Edit(ctx, owner, repo, number, req)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitHubIssueService) Reopen(ctx context.Context, owner, repo string, number int) error {
	state := "open"
	req := &github.IssueRequest{State: &state}
	_, resp, err := s.client.Issues.Edit(ctx, owner, repo, number, req)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitHubIssueService) Delete(_ context.Context, _, _ string, _ int) error {
	// GitHub doesn't support deleting issues via the API.
	return forge.ErrNotSupported
}

func (s *gitHubIssueService) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*forge.Comment, error) {
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

func (s *gitHubIssueService) ListComments(ctx context.Context, owner, repo string, number int) ([]forge.Comment, error) {
	var all []forge.Comment
	ghOpts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
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

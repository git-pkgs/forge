package gitea

import (
	"context"
	"fmt"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

type giteaIssueService struct {
	client *gitea.Client
}

func (f *giteaForge) Issues() forge.IssueService {
	return &giteaIssueService{client: f.client}
}

func convertGiteaIssue(i *gitea.Issue) forge.Issue {
	result := forge.Issue{
		Number:  int(i.Index),
		Title:   i.Title,
		Body:    i.Body,
		Locked:  i.IsLocked,
		HTMLURL: i.HTMLURL,
	}

	switch i.State {
	case gitea.StateOpen:
		result.State = "open"
	case gitea.StateClosed:
		result.State = "closed"
	default:
		result.State = string(i.State)
	}

	if i.Poster != nil {
		result.Author = forge.User{
			Login:     i.Poster.UserName,
			AvatarURL: i.Poster.AvatarURL,
		}
	}

	for _, a := range i.Assignees {
		result.Assignees = append(result.Assignees, forge.User{
			Login:     a.UserName,
			AvatarURL: a.AvatarURL,
		})
	}

	for _, l := range i.Labels {
		result.Labels = append(result.Labels, forge.Label{
			Name:        l.Name,
			Color:       l.Color,
			Description: l.Description,
		})
	}

	if i.Milestone != nil {
		result.Milestone = &forge.Milestone{
			Title:       i.Milestone.Title,
			Number:      int(i.Milestone.ID),
			Description: i.Milestone.Description,
			State:       string(i.Milestone.State),
		}
		if i.Milestone.Deadline != nil && !i.Milestone.Deadline.IsZero() {
			t := *i.Milestone.Deadline
			result.Milestone.DueDate = &t
		}
	}

	result.Comments = i.Comments

	if !i.Created.IsZero() {
		result.CreatedAt = i.Created
	}
	if !i.Updated.IsZero() {
		result.UpdatedAt = i.Updated
	}
	if i.Closed != nil && !i.Closed.IsZero() {
		result.ClosedAt = i.Closed
	}

	return result
}

func convertGiteaComment(c *gitea.Comment) forge.Comment {
	result := forge.Comment{
		ID:      c.ID,
		Body:    c.Body,
		HTMLURL: c.HTMLURL,
	}
	if c.Poster != nil {
		result.Author = forge.User{
			Login:     c.Poster.UserName,
			AvatarURL: c.Poster.AvatarURL,
		}
	}
	if !c.Created.IsZero() {
		result.CreatedAt = c.Created
	}
	if !c.Updated.IsZero() {
		result.UpdatedAt = c.Updated
	}
	return result
}

func (s *giteaIssueService) Get(ctx context.Context, owner, repo string, number int) (*forge.Issue, error) {
	i, resp, err := s.client.GetIssue(owner, repo, int64(number))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaIssue(i)
	return &result, nil
}

func (s *giteaIssueService) List(ctx context.Context, owner, repo string, opts forge.ListIssueOpts) ([]forge.Issue, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	gOpts := gitea.ListIssueOption{
		ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
		Type:        gitea.IssueTypeIssue,
	}

	switch opts.State {
	case "open":
		gOpts.State = gitea.StateOpen
	case "closed":
		gOpts.State = gitea.StateClosed
	case "all":
		gOpts.State = gitea.StateAll
	default:
		gOpts.State = gitea.StateOpen
	}

	if len(opts.Labels) > 0 {
		gOpts.Labels = opts.Labels
	}

	var all []forge.Issue
	for {
		issues, resp, err := s.client.ListRepoIssues(owner, repo, gOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, i := range issues {
			all = append(all, convertGiteaIssue(i))
		}
		if len(issues) < perPage || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		gOpts.Page++
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *giteaIssueService) Create(ctx context.Context, owner, repo string, opts forge.CreateIssueOpts) (*forge.Issue, error) {
	gOpts := gitea.CreateIssueOption{
		Title: opts.Title,
		Body:  opts.Body,
	}
	if len(opts.Assignees) > 0 {
		gOpts.Assignees = opts.Assignees
	}
	if len(opts.Labels) > 0 {
		ids, err := s.resolveLabelIDs(owner, repo, opts.Labels)
		if err != nil {
			return nil, fmt.Errorf("resolving labels: %w", err)
		}
		gOpts.Labels = ids
	}

	i, resp, err := s.client.CreateIssue(owner, repo, gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaIssue(i)
	return &result, nil
}

func (s *giteaIssueService) Update(ctx context.Context, owner, repo string, number int, opts forge.UpdateIssueOpts) (*forge.Issue, error) {
	gOpts := gitea.EditIssueOption{}
	changed := false

	if opts.Title != nil {
		gOpts.Title = *opts.Title
		changed = true
	}
	if opts.Body != nil {
		gOpts.Body = opts.Body
		changed = true
	}
	if opts.Assignees != nil {
		gOpts.Assignees = opts.Assignees
		changed = true
	}

	if !changed {
		return s.Get(ctx, owner, repo, number)
	}

	i, resp, err := s.client.EditIssue(owner, repo, int64(number), gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaIssue(i)
	return &result, nil
}

func (s *giteaIssueService) Close(ctx context.Context, owner, repo string, number int) error {
	closed := gitea.StateClosed
	gOpts := gitea.EditIssueOption{
		State: &closed,
	}
	_, resp, err := s.client.EditIssue(owner, repo, int64(number), gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *giteaIssueService) Reopen(ctx context.Context, owner, repo string, number int) error {
	open := gitea.StateOpen
	gOpts := gitea.EditIssueOption{
		State: &open,
	}
	_, resp, err := s.client.EditIssue(owner, repo, int64(number), gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *giteaIssueService) Delete(ctx context.Context, owner, repo string, number int) error {
	resp, err := s.client.DeleteIssue(owner, repo, int64(number))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *giteaIssueService) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*forge.Comment, error) {
	c, resp, err := s.client.CreateIssueComment(owner, repo, int64(number), gitea.CreateIssueCommentOption{
		Body: body,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaComment(c)
	return &result, nil
}

func (s *giteaIssueService) resolveLabelIDs(owner, repo string, names []string) ([]int64, error) {
	return resolveLabelIDs(s.client, owner, repo, names)
}

func (s *giteaIssueService) ListComments(ctx context.Context, owner, repo string, number int) ([]forge.Comment, error) {
	var all []forge.Comment
	page := 1
	for {
		comments, resp, err := s.client.ListIssueComments(owner, repo, int64(number), gitea.ListIssueCommentOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: 50},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, c := range comments {
			all = append(all, convertGiteaComment(c))
		}
		if len(comments) < 50 {
			break
		}
		page++
	}
	return all, nil
}

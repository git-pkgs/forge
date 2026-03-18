package gitlab

import (
	"context"
	"fmt"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const (
	stateAll = "all"
)

type gitLabIssueService struct {
	client *gitlab.Client
}

func (f *gitLabForge) Issues() forge.IssueService {
	return &gitLabIssueService{client: f.client}
}

func convertGitLabIssue(i *gitlab.Issue) forge.Issue {
	result := forge.Issue{
		Number:  int(i.IID),
		Title:   i.Title,
		Body:    i.Description,
		State:   i.State,
		Locked:  i.DiscussionLocked,
		HTMLURL: i.WebURL,
	}

	if i.Author != nil {
		result.Author = forge.User{
			Login:     i.Author.Username,
			Name:      i.Author.Name,
			AvatarURL: i.Author.AvatarURL,
			HTMLURL:   i.Author.WebURL,
		}
	}

	for _, a := range i.Assignees {
		result.Assignees = append(result.Assignees, forge.User{
			Login:     a.Username,
			Name:      a.Name,
			AvatarURL: a.AvatarURL,
			HTMLURL:   a.WebURL,
		})
	}

	for _, l := range i.Labels {
		result.Labels = append(result.Labels, forge.Label{Name: l})
	}

	if i.Milestone != nil {
		result.Milestone = &forge.Milestone{
			Title:       i.Milestone.Title,
			Number:      int(i.Milestone.ID),
			Description: i.Milestone.Description,
			State:       i.Milestone.State,
		}
		if i.Milestone.DueDate != nil {
			t := time.Time(*i.Milestone.DueDate)
			result.Milestone.DueDate = &t
		}
	}

	result.Comments = int(i.UserNotesCount)

	if i.CreatedAt != nil {
		result.CreatedAt = *i.CreatedAt
	}
	if i.UpdatedAt != nil {
		result.UpdatedAt = *i.UpdatedAt
	}
	if i.ClosedAt != nil {
		result.ClosedAt = i.ClosedAt
	}

	return result
}

func convertGitLabNote(n *gitlab.Note) forge.Comment {
	result := forge.Comment{
		ID:   n.ID,
		Body: n.Body,
		Author: forge.User{
			Login:     n.Author.Username,
			Name:      n.Author.Name,
			AvatarURL: n.Author.AvatarURL,
			HTMLURL:   n.Author.WebURL,
		},
	}
	if n.CreatedAt != nil {
		result.CreatedAt = *n.CreatedAt
	}
	if n.UpdatedAt != nil {
		result.UpdatedAt = *n.UpdatedAt
	}
	return result
}

func (s *gitLabIssueService) Get(ctx context.Context, owner, repo string, number int) (*forge.Issue, error) {
	pid := owner + "/" + repo
	i, resp, err := s.client.Issues.GetIssue(pid, int64(number))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabIssue(i)
	return &result, nil
}

func (s *gitLabIssueService) List(ctx context.Context, owner, repo string, opts forge.ListIssueOpts) ([]forge.Issue, error) {
	pid := owner + "/" + repo
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	glOpts := &gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage), Page: int64(page)},
	}

	if opts.State != "" && opts.State != stateAll {
		glOpts.State = gitlab.Ptr(opts.State)
	}
	if opts.Assignee != "" {
		glOpts.AssigneeUsername = gitlab.Ptr(opts.Assignee)
	}
	if opts.Author != "" {
		glOpts.AuthorUsername = gitlab.Ptr(opts.Author)
	}
	if len(opts.Labels) > 0 {
		lbls := gitlab.LabelOptions(opts.Labels)
		glOpts.Labels = &lbls
	}
	if opts.Sort != "" {
		glOpts.OrderBy = gitlab.Ptr(opts.Sort)
	}
	if opts.Order != "" {
		glOpts.Sort = gitlab.Ptr(opts.Order)
	}

	var all []forge.Issue
	for {
		issues, resp, err := s.client.Issues.ListProjectIssues(pid, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, i := range issues {
			all = append(all, convertGitLabIssue(i))
		}
		if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		glOpts.Page = int64(resp.NextPage)
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *gitLabIssueService) Create(ctx context.Context, owner, repo string, opts forge.CreateIssueOpts) (*forge.Issue, error) {
	pid := owner + "/" + repo
	glOpts := &gitlab.CreateIssueOptions{
		Title:       gitlab.Ptr(opts.Title),
		Description: gitlab.Ptr(opts.Body),
	}
	if len(opts.Assignees) > 0 {
		return nil, fmt.Errorf("GitLab requires assignee IDs, not usernames; assignees cannot be set by username on create")
	}
	if len(opts.Labels) > 0 {
		lbls := gitlab.LabelOptions(opts.Labels)
		glOpts.Labels = &lbls
	}

	i, resp, err := s.client.Issues.CreateIssue(pid, glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabIssue(i)
	return &result, nil
}

func (s *gitLabIssueService) Update(ctx context.Context, owner, repo string, number int, opts forge.UpdateIssueOpts) (*forge.Issue, error) {
	pid := owner + "/" + repo
	glOpts := &gitlab.UpdateIssueOptions{}
	changed := false

	if opts.Title != nil {
		glOpts.Title = opts.Title
		changed = true
	}
	if opts.Body != nil {
		glOpts.Description = opts.Body
		changed = true
	}
	if opts.Labels != nil {
		lbls := gitlab.LabelOptions(opts.Labels)
		glOpts.Labels = &lbls
		changed = true
	}

	if !changed {
		return s.Get(ctx, owner, repo, number)
	}

	i, resp, err := s.client.Issues.UpdateIssue(pid, int64(number), glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabIssue(i)
	return &result, nil
}

func (s *gitLabIssueService) Close(ctx context.Context, owner, repo string, number int) error {
	pid := owner + "/" + repo
	glOpts := &gitlab.UpdateIssueOptions{
		StateEvent: gitlab.Ptr("close"),
	}
	_, resp, err := s.client.Issues.UpdateIssue(pid, int64(number), glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabIssueService) Reopen(ctx context.Context, owner, repo string, number int) error {
	pid := owner + "/" + repo
	glOpts := &gitlab.UpdateIssueOptions{
		StateEvent: gitlab.Ptr("reopen"),
	}
	_, resp, err := s.client.Issues.UpdateIssue(pid, int64(number), glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabIssueService) Delete(ctx context.Context, owner, repo string, number int) error {
	pid := owner + "/" + repo
	resp, err := s.client.Issues.DeleteIssue(pid, int64(number))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabIssueService) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*forge.Comment, error) {
	pid := owner + "/" + repo
	n, resp, err := s.client.Notes.CreateIssueNote(pid, int64(number), &gitlab.CreateIssueNoteOptions{
		Body: gitlab.Ptr(body),
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabNote(n)
	return &result, nil
}

func (s *gitLabIssueService) ListComments(ctx context.Context, owner, repo string, number int) ([]forge.Comment, error) {
	pid := owner + "/" + repo
	var all []forge.Comment
	glOpts := &gitlab.ListIssueNotesOptions{
		ListOptions: gitlab.ListOptions{PerPage: defaultPageSize},
	}
	for {
		notes, resp, err := s.client.Notes.ListIssueNotes(pid, int64(number), glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, n := range notes {
			// Skip system notes (auto-generated by GitLab)
			if n.System {
				continue
			}
			all = append(all, convertGitLabNote(n))
		}
		if resp.NextPage == 0 {
			break
		}
		glOpts.Page = int64(resp.NextPage)
	}
	return all, nil
}

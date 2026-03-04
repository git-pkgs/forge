package forges

import (
	"context"
	"net/http"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabPRService struct {
	client *gitlab.Client
}

func (f *gitLabForge) PullRequests() PullRequestService {
	return &gitLabPRService{client: f.client}
}

func convertGitLabMR(mr *gitlab.MergeRequest) PullRequest {
	result := PullRequest{
		Number:       int(mr.IID),
		Title:        mr.Title,
		Body:         mr.Description,
		State:        mr.State, // "opened", "closed", "merged"
		Draft:        mr.Draft,
		Head:         mr.SourceBranch,
		Base:         mr.TargetBranch,
		Merged:       mr.State == "merged",
		Comments:     int(mr.UserNotesCount),
		// ChangesCount is a string in the GitLab API
		HTMLURL:      mr.WebURL,
	}

	// Normalize "opened" to "open"
	if result.State == "opened" {
		result.State = "open"
	}

	if mr.Author != nil {
		result.Author = User{
			Login:     mr.Author.Username,
			Name:      mr.Author.Name,
			AvatarURL: mr.Author.AvatarURL,
			HTMLURL:   mr.Author.WebURL,
		}
	}

	for _, a := range mr.Assignees {
		result.Assignees = append(result.Assignees, User{
			Login:     a.Username,
			Name:      a.Name,
			AvatarURL: a.AvatarURL,
			HTMLURL:   a.WebURL,
		})
	}

	for _, r := range mr.Reviewers {
		result.Reviewers = append(result.Reviewers, User{
			Login:     r.Username,
			Name:      r.Name,
			AvatarURL: r.AvatarURL,
			HTMLURL:   r.WebURL,
		})
	}

	for _, l := range mr.Labels {
		result.Labels = append(result.Labels, Label{Name: l})
	}

	if mr.Milestone != nil {
		result.Milestone = &Milestone{
			Title:       mr.Milestone.Title,
			Number:      int(mr.Milestone.ID),
			Description: mr.Milestone.Description,
			State:       mr.Milestone.State,
		}
		if mr.Milestone.DueDate != nil {
			t := time.Time(*mr.Milestone.DueDate)
			result.Milestone.DueDate = &t
		}
	}

	if mr.MergedBy != nil {
		result.MergedBy = &User{
			Login:     mr.MergedBy.Username,
			Name:      mr.MergedBy.Name,
			AvatarURL: mr.MergedBy.AvatarURL,
			HTMLURL:   mr.MergedBy.WebURL,
		}
	}

	if mr.CreatedAt != nil {
		result.CreatedAt = *mr.CreatedAt
	}
	if mr.UpdatedAt != nil {
		result.UpdatedAt = *mr.UpdatedAt
	}
	if mr.MergedAt != nil {
		result.MergedAt = mr.MergedAt
	}
	if mr.ClosedAt != nil {
		result.ClosedAt = mr.ClosedAt
	}

	return result
}

func convertBasicGitLabMR(mr *gitlab.BasicMergeRequest) PullRequest {
	result := PullRequest{
		Number:  int(mr.IID),
		Title:   mr.Title,
		Body:    mr.Description,
		State:   mr.State,
		Draft:   mr.Draft,
		Head:    mr.SourceBranch,
		Base:    mr.TargetBranch,
		Merged:  mr.State == "merged",
		HTMLURL: mr.WebURL,
	}

	if result.State == "opened" {
		result.State = "open"
	}

	if mr.Author != nil {
		result.Author = User{
			Login:     mr.Author.Username,
			Name:      mr.Author.Name,
			AvatarURL: mr.Author.AvatarURL,
			HTMLURL:   mr.Author.WebURL,
		}
	}

	for _, a := range mr.Assignees {
		result.Assignees = append(result.Assignees, User{
			Login:     a.Username,
			Name:      a.Name,
			AvatarURL: a.AvatarURL,
			HTMLURL:   a.WebURL,
		})
	}

	for _, r := range mr.Reviewers {
		result.Reviewers = append(result.Reviewers, User{
			Login:     r.Username,
			Name:      r.Name,
			AvatarURL: r.AvatarURL,
			HTMLURL:   r.WebURL,
		})
	}

	for _, l := range mr.Labels {
		result.Labels = append(result.Labels, Label{Name: l})
	}

	if mr.CreatedAt != nil {
		result.CreatedAt = *mr.CreatedAt
	}
	if mr.UpdatedAt != nil {
		result.UpdatedAt = *mr.UpdatedAt
	}
	if mr.MergedAt != nil {
		result.MergedAt = mr.MergedAt
	}
	if mr.ClosedAt != nil {
		result.ClosedAt = mr.ClosedAt
	}

	return result
}

func (s *gitLabPRService) Get(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	pid := owner + "/" + repo
	mr, resp, err := s.client.MergeRequests.GetMergeRequest(pid, int64(number), nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabMR(mr)
	return &result, nil
}

func (s *gitLabPRService) List(ctx context.Context, owner, repo string, opts ListPROpts) ([]PullRequest, error) {
	pid := owner + "/" + repo
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	glOpts := &gitlab.ListProjectMergeRequestsOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage), Page: int64(page)},
	}

	if opts.State != "" && opts.State != "all" {
		// GitLab uses "opened" not "open"
		state := opts.State
		if state == "open" {
			state = "opened"
		}
		glOpts.State = gitlab.Ptr(state)
	}
	if opts.Author != "" {
		glOpts.AuthorUsername = gitlab.Ptr(opts.Author)
	}
	// GitLab MR list uses AssigneeID (not username), so we skip assignee filtering here
	if len(opts.Labels) > 0 {
		lbls := gitlab.LabelOptions(opts.Labels)
		glOpts.Labels = &lbls
	}
	if opts.Base != "" {
		glOpts.TargetBranch = gitlab.Ptr(opts.Base)
	}
	if opts.Head != "" {
		glOpts.SourceBranch = gitlab.Ptr(opts.Head)
	}
	if opts.Sort != "" {
		glOpts.OrderBy = gitlab.Ptr(opts.Sort)
	}
	if opts.Order != "" {
		glOpts.Sort = gitlab.Ptr(opts.Order)
	}

	var all []PullRequest
	for {
		mrs, resp, err := s.client.MergeRequests.ListProjectMergeRequests(pid, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		for _, mr := range mrs {
			all = append(all, convertBasicGitLabMR(mr))
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

func (s *gitLabPRService) Create(ctx context.Context, owner, repo string, opts CreatePROpts) (*PullRequest, error) {
	pid := owner + "/" + repo
	glOpts := &gitlab.CreateMergeRequestOptions{
		Title:        gitlab.Ptr(opts.Title),
		Description:  gitlab.Ptr(opts.Body),
		SourceBranch: gitlab.Ptr(opts.Head),
		TargetBranch: gitlab.Ptr(opts.Base),
	}

	if opts.Draft {
		glOpts.Title = gitlab.Ptr("Draft: " + opts.Title)
	}

	if len(opts.Labels) > 0 {
		lbls := gitlab.LabelOptions(opts.Labels)
		glOpts.Labels = &lbls
	}

	mr, resp, err := s.client.MergeRequests.CreateMergeRequest(pid, glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabMR(mr)
	return &result, nil
}

func (s *gitLabPRService) Update(ctx context.Context, owner, repo string, number int, opts UpdatePROpts) (*PullRequest, error) {
	pid := owner + "/" + repo
	glOpts := &gitlab.UpdateMergeRequestOptions{}
	changed := false

	if opts.Title != nil {
		glOpts.Title = opts.Title
		changed = true
	}
	if opts.Body != nil {
		glOpts.Description = opts.Body
		changed = true
	}
	if opts.Base != nil {
		glOpts.TargetBranch = opts.Base
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

	mr, resp, err := s.client.MergeRequests.UpdateMergeRequest(pid, int64(number), glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabMR(mr)
	return &result, nil
}

func (s *gitLabPRService) Close(ctx context.Context, owner, repo string, number int) error {
	pid := owner + "/" + repo
	glOpts := &gitlab.UpdateMergeRequestOptions{
		StateEvent: gitlab.Ptr("close"),
	}
	_, resp, err := s.client.MergeRequests.UpdateMergeRequest(pid, int64(number), glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabPRService) Reopen(ctx context.Context, owner, repo string, number int) error {
	pid := owner + "/" + repo
	glOpts := &gitlab.UpdateMergeRequestOptions{
		StateEvent: gitlab.Ptr("reopen"),
	}
	_, resp, err := s.client.MergeRequests.UpdateMergeRequest(pid, int64(number), glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabPRService) Merge(ctx context.Context, owner, repo string, number int, opts MergePROpts) error {
	pid := owner + "/" + repo
	glOpts := &gitlab.AcceptMergeRequestOptions{}

	if opts.Message != "" {
		glOpts.MergeCommitMessage = gitlab.Ptr(opts.Message)
	}
	if opts.Method == "squash" {
		glOpts.Squash = gitlab.Ptr(true)
		if opts.Message != "" {
			glOpts.SquashCommitMessage = gitlab.Ptr(opts.Message)
		}
	}
	if opts.Delete {
		glOpts.ShouldRemoveSourceBranch = gitlab.Ptr(true)
	}

	_, resp, err := s.client.MergeRequests.AcceptMergeRequest(pid, int64(number), glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabPRService) Diff(ctx context.Context, owner, repo string, number int) (string, error) {
	pid := owner + "/" + repo
	raw, resp, err := s.client.MergeRequests.ShowMergeRequestRawDiffs(pid, int64(number), nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return "", ErrNotFound
		}
		return "", err
	}
	return string(raw), nil
}

func (s *gitLabPRService) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*Comment, error) {
	pid := owner + "/" + repo
	n, resp, err := s.client.Notes.CreateMergeRequestNote(pid, int64(number), &gitlab.CreateMergeRequestNoteOptions{
		Body: gitlab.Ptr(body),
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabNote(n)
	return &result, nil
}

func (s *gitLabPRService) ListComments(ctx context.Context, owner, repo string, number int) ([]Comment, error) {
	pid := owner + "/" + repo
	var all []Comment
	glOpts := &gitlab.ListMergeRequestNotesOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}
	for {
		notes, resp, err := s.client.Notes.ListMergeRequestNotes(pid, int64(number), glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		for _, n := range notes {
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

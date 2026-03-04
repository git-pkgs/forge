package gitea

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

type giteaPRService struct {
	client *gitea.Client
}

func (f *giteaForge) PullRequests() forge.PullRequestService {
	return &giteaPRService{client: f.client}
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func convertGiteaPR(pr *gitea.PullRequest) forge.PullRequest {
	result := forge.PullRequest{
		Number:       int(pr.Index),
		Title:        pr.Title,
		Body:         pr.Body,
		Draft:        pr.Draft,
		Mergeable:    pr.Mergeable,
		Merged:       pr.HasMerged,
		Additions:    derefInt(pr.Additions),
		Deletions:    derefInt(pr.Deletions),
		ChangedFiles: derefInt(pr.ChangedFiles),
		HTMLURL:      pr.HTMLURL,
		DiffURL:      pr.DiffURL,
	}

	if pr.HasMerged {
		result.State = "merged"
	} else if pr.State == gitea.StateClosed {
		result.State = "closed"
	} else {
		result.State = "open"
	}

	if pr.Head != nil {
		result.Head = pr.Head.Name
	}
	if pr.Base != nil {
		result.Base = pr.Base.Name
	}

	if pr.Poster != nil {
		result.Author = forge.User{
			Login:     pr.Poster.UserName,
			AvatarURL: pr.Poster.AvatarURL,
		}
	}

	for _, a := range pr.Assignees {
		result.Assignees = append(result.Assignees, forge.User{
			Login:     a.UserName,
			AvatarURL: a.AvatarURL,
		})
	}

	for _, l := range pr.Labels {
		result.Labels = append(result.Labels, forge.Label{
			Name:        l.Name,
			Color:       l.Color,
			Description: l.Description,
		})
	}

	if pr.Milestone != nil {
		result.Milestone = &forge.Milestone{
			Title:       pr.Milestone.Title,
			Number:      int(pr.Milestone.ID),
			Description: pr.Milestone.Description,
			State:       string(pr.Milestone.State),
		}
	}

	if pr.MergedBy != nil {
		result.MergedBy = &forge.User{
			Login:     pr.MergedBy.UserName,
			AvatarURL: pr.MergedBy.AvatarURL,
		}
	}

	if pr.Created != nil && !pr.Created.IsZero() {
		result.CreatedAt = *pr.Created
	}
	if pr.Updated != nil && !pr.Updated.IsZero() {
		result.UpdatedAt = *pr.Updated
	}
	if pr.Merged != nil && !pr.Merged.IsZero() {
		result.MergedAt = pr.Merged
	}
	if pr.Closed != nil && !pr.Closed.IsZero() {
		result.ClosedAt = pr.Closed
	}

	return result
}

func (s *giteaPRService) Get(ctx context.Context, owner, repo string, number int) (*forge.PullRequest, error) {
	pr, resp, err := s.client.GetPullRequest(owner, repo, int64(number))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaPR(pr)
	return &result, nil
}

func (s *giteaPRService) List(ctx context.Context, owner, repo string, opts forge.ListPROpts) ([]forge.PullRequest, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	gOpts := gitea.ListPullRequestsOptions{
		ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
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

	if opts.Sort != "" {
		gOpts.Sort = opts.Sort
	}

	var all []forge.PullRequest
	for {
		prs, resp, err := s.client.ListRepoPullRequests(owner, repo, gOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, pr := range prs {
			all = append(all, convertGiteaPR(pr))
		}
		if len(prs) < perPage || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		gOpts.Page++
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *giteaPRService) Create(ctx context.Context, owner, repo string, opts forge.CreatePROpts) (*forge.PullRequest, error) {
	gOpts := gitea.CreatePullRequestOption{
		Title: opts.Title,
		Body:  opts.Body,
		Head:  opts.Head,
		Base:  opts.Base,
	}
	if len(opts.Assignees) > 0 {
		gOpts.Assignees = opts.Assignees
	}
	// Gitea requires label IDs, not names.

	pr, resp, err := s.client.CreatePullRequest(owner, repo, gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaPR(pr)
	return &result, nil
}

func (s *giteaPRService) Update(ctx context.Context, owner, repo string, number int, opts forge.UpdatePROpts) (*forge.PullRequest, error) {
	gOpts := gitea.EditPullRequestOption{}
	changed := false

	if opts.Title != nil {
		gOpts.Title = *opts.Title
		changed = true
	}
	if opts.Body != nil {
		gOpts.Body = opts.Body
		changed = true
	}
	if opts.Base != nil {
		gOpts.Base = *opts.Base
		changed = true
	}
	if opts.Assignees != nil {
		gOpts.Assignees = opts.Assignees
		changed = true
	}

	if !changed {
		return s.Get(ctx, owner, repo, number)
	}

	pr, resp, err := s.client.EditPullRequest(owner, repo, int64(number), gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaPR(pr)
	return &result, nil
}

func (s *giteaPRService) Close(ctx context.Context, owner, repo string, number int) error {
	closed := gitea.StateClosed
	gOpts := gitea.EditPullRequestOption{
		State: &closed,
	}
	_, resp, err := s.client.EditPullRequest(owner, repo, int64(number), gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *giteaPRService) Reopen(ctx context.Context, owner, repo string, number int) error {
	open := gitea.StateOpen
	gOpts := gitea.EditPullRequestOption{
		State: &open,
	}
	_, resp, err := s.client.EditPullRequest(owner, repo, int64(number), gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *giteaPRService) Merge(ctx context.Context, owner, repo string, number int, opts forge.MergePROpts) error {
	gOpts := gitea.MergePullRequestOption{
		Style:   gitea.MergeStyle(opts.Method),
		Message: opts.Message,
		Title:   opts.Title,
	}
	if gOpts.Style == "" {
		gOpts.Style = gitea.MergeStyleMerge
	}
	if opts.Delete {
		gOpts.DeleteBranchAfterMerge = true
	}

	_, resp, err := s.client.MergePullRequest(owner, repo, int64(number), gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *giteaPRService) Diff(ctx context.Context, owner, repo string, number int) (string, error) {
	raw, resp, err := s.client.GetPullRequestDiff(owner, repo, int64(number), gitea.PullRequestDiffOptions{})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return "", forge.ErrNotFound
		}
		return "", err
	}
	return string(raw), nil
}

func (s *giteaPRService) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*forge.Comment, error) {
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

func (s *giteaPRService) ListComments(ctx context.Context, owner, repo string, number int) ([]forge.Comment, error) {
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

package bitbucket

import (
	"context"
	"fmt"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"time"
)

type bitbucketIssueService struct {
	token      string
	httpClient *http.Client
}

func (f *bitbucketForge) Issues() forge.IssueService {
	return &bitbucketIssueService{token: f.token, httpClient: f.httpClient}
}

type bbIssue struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
	State    string `json:"state"` // new, open, resolved, closed, etc.
	Priority string `json:"priority"`
	Reporter *struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Links       struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
		} `json:"links"`
	} `json:"reporter"`
	Assignee *struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Links       struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
		} `json:"links"`
	} `json:"assignee"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
}

type bbIssuesResponse struct {
	Values []bbIssue `json:"values"`
	Next   string    `json:"next"`
}

type bbComment struct {
	ID      int `json:"id"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
	User *struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Links       struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
			Avatar struct {
				Href string `json:"href"`
			} `json:"avatar"`
		} `json:"links"`
	} `json:"user"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
}

type bbCommentsResponse struct {
	Values []bbComment `json:"values"`
	Next   string      `json:"next"`
}

func convertBitbucketIssue(bb bbIssue) forge.Issue {
	result := forge.Issue{
		Number:  bb.ID,
		Title:   bb.Title,
		Body:    bb.Content.Raw,
		HTMLURL: bb.Links.HTML.Href,
	}

	// Normalize Bitbucket states to open/closed
	switch bb.State {
	case "new", "open":
		result.State = "open"
	default:
		result.State = "closed"
	}

	if bb.Reporter != nil {
		result.Author = forge.User{
			Login:     bb.Reporter.Username,
			AvatarURL: bb.Reporter.Links.Avatar.Href,
			HTMLURL:   bb.Reporter.Links.HTML.Href,
		}
	}

	if bb.Assignee != nil {
		result.Assignees = []forge.User{{
			Login:     bb.Assignee.Username,
			AvatarURL: bb.Assignee.Links.Avatar.Href,
			HTMLURL:   bb.Assignee.Links.HTML.Href,
		}}
	}

	if t, err := time.Parse(time.RFC3339, bb.CreatedOn); err == nil {
		result.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, bb.UpdatedOn); err == nil {
		result.UpdatedAt = t
	}

	return result
}

func convertBitbucketComment(bb bbComment) forge.Comment {
	result := forge.Comment{
		ID:      int64(bb.ID),
		Body:    bb.Content.Raw,
		HTMLURL: bb.Links.HTML.Href,
	}
	if bb.User != nil {
		result.Author = forge.User{
			Login:     bb.User.Username,
			AvatarURL: bb.User.Links.Avatar.Href,
			HTMLURL:   bb.User.Links.HTML.Href,
		}
	}
	if t, err := time.Parse(time.RFC3339, bb.CreatedOn); err == nil {
		result.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, bb.UpdatedOn); err == nil {
		result.UpdatedAt = t
	}
	return result
}

func (s *bitbucketIssueService) doJSON(ctx context.Context, method, url string, body any, v any) error {
	rs := &bitbucketRepoService{token: s.token, httpClient: s.httpClient}
	return rs.doJSON(ctx, method, url, body, v)
}

func (s *bitbucketIssueService) Get(ctx context.Context, owner, repo string, number int) (*forge.Issue, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/issues/%d", bitbucketAPI, owner, repo, number)
	var bb bbIssue
	if err := s.doJSON(ctx, http.MethodGet, url, nil, &bb); err != nil {
		return nil, err
	}
	result := convertBitbucketIssue(bb)
	return &result, nil
}

func (s *bitbucketIssueService) List(ctx context.Context, owner, repo string, opts forge.ListIssueOpts) ([]forge.Issue, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}

	url := fmt.Sprintf("%s/repositories/%s/%s/issues?pagelen=%d", bitbucketAPI, owner, repo, perPage)

	// Bitbucket uses q= query parameter for filtering
	switch opts.State {
	case "open":
		url += "&q=state+%3D+%22new%22+OR+state+%3D+%22open%22"
	case "closed":
		url += "&q=state+%3D+%22resolved%22+OR+state+%3D+%22closed%22"
	}

	var all []forge.Issue
	for url != "" {
		var page bbIssuesResponse
		if err := s.doJSON(ctx, http.MethodGet, url, nil, &page); err != nil {
			return nil, err
		}
		for _, bb := range page.Values {
			all = append(all, convertBitbucketIssue(bb))
		}
		url = page.Next
		if opts.Limit > 0 && len(all) >= opts.Limit {
			break
		}
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *bitbucketIssueService) Create(ctx context.Context, owner, repo string, opts forge.CreateIssueOpts) (*forge.Issue, error) {
	body := map[string]any{
		"title": opts.Title,
		"content": map[string]string{
			"raw": opts.Body,
		},
	}
	if len(opts.Assignees) > 0 {
		body["assignee"] = map[string]string{"username": opts.Assignees[0]}
	}

	url := fmt.Sprintf("%s/repositories/%s/%s/issues", bitbucketAPI, owner, repo)
	var bb bbIssue
	if err := s.doJSON(ctx, http.MethodPost, url, body, &bb); err != nil {
		return nil, err
	}
	result := convertBitbucketIssue(bb)
	return &result, nil
}

func (s *bitbucketIssueService) Update(ctx context.Context, owner, repo string, number int, opts forge.UpdateIssueOpts) (*forge.Issue, error) {
	body := map[string]any{}
	if opts.Title != nil {
		body["title"] = *opts.Title
	}
	if opts.Body != nil {
		body["content"] = map[string]string{"raw": *opts.Body}
	}
	if len(opts.Assignees) > 0 {
		body["assignee"] = map[string]string{"username": opts.Assignees[0]}
	}

	if len(body) == 0 {
		return s.Get(ctx, owner, repo, number)
	}

	url := fmt.Sprintf("%s/repositories/%s/%s/issues/%d", bitbucketAPI, owner, repo, number)
	var bb bbIssue
	if err := s.doJSON(ctx, http.MethodPut, url, body, &bb); err != nil {
		return nil, err
	}
	result := convertBitbucketIssue(bb)
	return &result, nil
}

func (s *bitbucketIssueService) Close(ctx context.Context, owner, repo string, number int) error {
	body := map[string]any{"state": "resolved"}
	url := fmt.Sprintf("%s/repositories/%s/%s/issues/%d", bitbucketAPI, owner, repo, number)
	return s.doJSON(ctx, http.MethodPut, url, body, nil)
}

func (s *bitbucketIssueService) Reopen(ctx context.Context, owner, repo string, number int) error {
	body := map[string]any{"state": "open"}
	url := fmt.Sprintf("%s/repositories/%s/%s/issues/%d", bitbucketAPI, owner, repo, number)
	return s.doJSON(ctx, http.MethodPut, url, body, nil)
}

func (s *bitbucketIssueService) Delete(ctx context.Context, owner, repo string, number int) error {
	url := fmt.Sprintf("%s/repositories/%s/%s/issues/%d", bitbucketAPI, owner, repo, number)
	return s.doJSON(ctx, http.MethodDelete, url, nil, nil)
}

func (s *bitbucketIssueService) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*forge.Comment, error) {
	reqBody := map[string]any{
		"content": map[string]string{"raw": body},
	}
	url := fmt.Sprintf("%s/repositories/%s/%s/issues/%d/comments", bitbucketAPI, owner, repo, number)
	var bb bbComment
	if err := s.doJSON(ctx, http.MethodPost, url, reqBody, &bb); err != nil {
		return nil, err
	}
	result := convertBitbucketComment(bb)
	return &result, nil
}

func (s *bitbucketIssueService) ListComments(ctx context.Context, owner, repo string, number int) ([]forge.Comment, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/issues/%d/comments?pagelen=100", bitbucketAPI, owner, repo, number)
	var all []forge.Comment
	for url != "" {
		var page bbCommentsResponse
		if err := s.doJSON(ctx, http.MethodGet, url, nil, &page); err != nil {
			return nil, err
		}
		for _, bb := range page.Values {
			all = append(all, convertBitbucketComment(bb))
		}
		url = page.Next
	}
	return all, nil
}

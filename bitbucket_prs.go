package forges

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type bitbucketPRService struct {
	token      string
	httpClient *http.Client
}

func (f *bitbucketForge) PullRequests() PullRequestService {
	return &bitbucketPRService{token: f.token, httpClient: f.httpClient}
}

type bbPullRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"` // OPEN, MERGED, DECLINED, SUPERSEDED
	Source      struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"destination"`
	Author *struct {
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
	} `json:"author"`
	Reviewers []struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	} `json:"reviewers"`
	MergeCommit *struct {
		Hash string `json:"hash"`
	} `json:"merge_commit"`
	ClosedBy *struct {
		Username string `json:"username"`
	} `json:"closed_by"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Diff struct {
			Href string `json:"href"`
		} `json:"diff"`
	} `json:"links"`
	CommentCount int    `json:"comment_count"`
	CreatedOn    string `json:"created_on"`
	UpdatedOn    string `json:"updated_on"`
}

type bbPRsResponse struct {
	Values []bbPullRequest `json:"values"`
	Next   string          `json:"next"`
}

func convertBitbucketPR(bb bbPullRequest) PullRequest {
	result := PullRequest{
		Number:   bb.ID,
		Title:    bb.Title,
		Body:     bb.Description,
		Head:     bb.Source.Branch.Name,
		Base:     bb.Destination.Branch.Name,
		Comments: bb.CommentCount,
		HTMLURL:  bb.Links.HTML.Href,
		DiffURL:  bb.Links.Diff.Href,
	}

	switch bb.State {
	case "OPEN":
		result.State = "open"
	case "MERGED":
		result.State = "merged"
		result.Merged = true
	default:
		result.State = "closed"
	}

	if bb.Author != nil {
		result.Author = User{
			Login:     bb.Author.Username,
			AvatarURL: bb.Author.Links.Avatar.Href,
			HTMLURL:   bb.Author.Links.HTML.Href,
		}
	}

	for _, r := range bb.Reviewers {
		result.Reviewers = append(result.Reviewers, User{
			Login: r.Username,
		})
	}

	if t, err := time.Parse(time.RFC3339, bb.CreatedOn); err == nil {
		result.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, bb.UpdatedOn); err == nil {
		result.UpdatedAt = t
	}

	return result
}

func (s *bitbucketPRService) doJSON(ctx context.Context, method, url string, body any, v any) error {
	rs := &bitbucketRepoService{token: s.token, httpClient: s.httpClient}
	return rs.doJSON(ctx, method, url, body, v)
}

func (s *bitbucketPRService) Get(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d", bitbucketAPI, owner, repo, number)
	var bb bbPullRequest
	if err := s.doJSON(ctx, http.MethodGet, url, nil, &bb); err != nil {
		return nil, err
	}
	result := convertBitbucketPR(bb)
	return &result, nil
}

func (s *bitbucketPRService) List(ctx context.Context, owner, repo string, opts ListPROpts) ([]PullRequest, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}

	state := "OPEN"
	switch opts.State {
	case "closed":
		state = "DECLINED"
	case "merged":
		state = "MERGED"
	case "all":
		state = ""
	}

	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests?pagelen=%d", bitbucketAPI, owner, repo, perPage)
	if state != "" {
		url += "&state=" + state
	}

	var all []PullRequest
	for url != "" {
		var page bbPRsResponse
		if err := s.doJSON(ctx, http.MethodGet, url, nil, &page); err != nil {
			return nil, err
		}
		for _, bb := range page.Values {
			all = append(all, convertBitbucketPR(bb))
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

func (s *bitbucketPRService) Create(ctx context.Context, owner, repo string, opts CreatePROpts) (*PullRequest, error) {
	body := map[string]any{
		"title":       opts.Title,
		"description": opts.Body,
		"source": map[string]any{
			"branch": map[string]string{"name": opts.Head},
		},
		"destination": map[string]any{
			"branch": map[string]string{"name": opts.Base},
		},
		"close_source_branch": false,
	}

	if len(opts.Reviewers) > 0 {
		reviewers := make([]map[string]string, len(opts.Reviewers))
		for i, r := range opts.Reviewers {
			reviewers[i] = map[string]string{"username": r}
		}
		body["reviewers"] = reviewers
	}

	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests", bitbucketAPI, owner, repo)
	var bb bbPullRequest
	if err := s.doJSON(ctx, http.MethodPost, url, body, &bb); err != nil {
		return nil, err
	}
	result := convertBitbucketPR(bb)
	return &result, nil
}

func (s *bitbucketPRService) Update(ctx context.Context, owner, repo string, number int, opts UpdatePROpts) (*PullRequest, error) {
	body := map[string]any{}
	if opts.Title != nil {
		body["title"] = *opts.Title
	}
	if opts.Body != nil {
		body["description"] = *opts.Body
	}
	if opts.Base != nil {
		body["destination"] = map[string]any{
			"branch": map[string]string{"name": *opts.Base},
		}
	}
	if opts.Reviewers != nil {
		reviewers := make([]map[string]string, len(opts.Reviewers))
		for i, r := range opts.Reviewers {
			reviewers[i] = map[string]string{"username": r}
		}
		body["reviewers"] = reviewers
	}

	if len(body) == 0 {
		return s.Get(ctx, owner, repo, number)
	}

	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d", bitbucketAPI, owner, repo, number)
	var bb bbPullRequest
	if err := s.doJSON(ctx, http.MethodPut, url, body, &bb); err != nil {
		return nil, err
	}
	result := convertBitbucketPR(bb)
	return &result, nil
}

func (s *bitbucketPRService) Close(ctx context.Context, owner, repo string, number int) error {
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/decline", bitbucketAPI, owner, repo, number)
	return s.doJSON(ctx, http.MethodPost, url, nil, nil)
}

func (s *bitbucketPRService) Reopen(ctx context.Context, owner, repo string, number int) error {
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d", bitbucketAPI, owner, repo, number)
	body := map[string]any{"state": "OPEN"}
	return s.doJSON(ctx, http.MethodPut, url, body, nil)
}

func (s *bitbucketPRService) Merge(ctx context.Context, owner, repo string, number int, opts MergePROpts) error {
	body := map[string]any{}
	if opts.Method != "" {
		body["merge_strategy"] = opts.Method
	}
	if opts.Message != "" {
		body["message"] = opts.Message
	}
	if opts.Delete {
		body["close_source_branch"] = true
	}

	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/merge", bitbucketAPI, owner, repo, number)
	return s.doJSON(ctx, http.MethodPost, url, body, nil)
}

func (s *bitbucketPRService) Diff(ctx context.Context, owner, repo string, number int) (string, error) {
	// Bitbucket returns diff as plain text; we use doJSON but it'll fail to decode.
	// Use a direct GET instead.
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/diff", bitbucketAPI, owner, repo, number)
	rs := &bitbucketRepoService{token: s.token, httpClient: s.httpClient}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if rs.token != "" {
		req.Header.Set("Authorization", "Bearer "+rs.token)
	}

	resp, err := rs.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", ErrNotFound
	}

	var buf []byte
	buf = make([]byte, 0, 1024*64)
	tmp := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if readErr != nil {
			break
		}
	}
	return string(buf), nil
}

func (s *bitbucketPRService) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*Comment, error) {
	reqBody := map[string]any{
		"content": map[string]string{"raw": body},
	}
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments", bitbucketAPI, owner, repo, number)
	var bb bbComment
	if err := s.doJSON(ctx, http.MethodPost, url, reqBody, &bb); err != nil {
		return nil, err
	}
	result := convertBitbucketComment(bb)
	return &result, nil
}

func (s *bitbucketPRService) ListComments(ctx context.Context, owner, repo string, number int) ([]Comment, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments?pagelen=100", bitbucketAPI, owner, repo, number)
	var all []Comment
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

package gerrit

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	forge "github.com/git-pkgs/forge"
)

type gerritPRService struct {
	forge *gerritForge
}

type gerritAccountInfo struct {
	ID       int    `json:"_account_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type gerritRevisionInfo struct {
	Number int    `json:"_number"`
	Ref    string `json:"ref"`
}

type gerritChangeInfo struct {
	ID                     string                        `json:"id"`
	Project                string                        `json:"project"`
	Branch                 string                        `json:"branch"`
	ChangeID               string                        `json:"change_id"`
	Subject                string                        `json:"subject"`
	Status                 string                        `json:"status"`
	Owner                  *gerritAccountInfo            `json:"owner"`
	Created                string                        `json:"created"`
	Updated                string                        `json:"updated"`
	Submitted              string                        `json:"submitted"`
	Insertions             int                           `json:"insertions"`
	Deletions              int                           `json:"deletions"`
	Number                 int                           `json:"_number"`
	CurrentRevision        string                        `json:"current_revision"`
	Revisions              map[string]gerritRevisionInfo `json:"revisions"`
	WorkInProgress         bool                          `json:"work_in_progress"`
	UnresolvedCommentCount int                           `json:"unresolved_comment_count"`
	Messages               []gerritChangeMessageInfo     `json:"messages"`
	MoreChanges            bool                          `json:"_more_changes"`
}

type gerritChangeMessageInfo struct {
	ID      string             `json:"id"`
	Author  *gerritAccountInfo `json:"author"`
	Date    string             `json:"date"`
	Message string             `json:"message"`
}

func (f *gerritForge) PullRequests() forge.PullRequestService {
	return &gerritPRService{forge: f}
}

func (s *gerritPRService) convertAccount(a *gerritAccountInfo) forge.User {
	if a == nil {
		return forge.User{}
	}
	login := a.Username
	if login == "" {
		login = a.Email
	}
	if login == "" && a.ID != 0 {
		login = strconv.Itoa(a.ID)
	}
	return forge.User{
		Login: login,
		Name:  a.Name,
		Email: a.Email,
	}
}

func (s *gerritPRService) convertChange(c gerritChangeInfo) forge.PullRequest {
	state := forge.NormalizePRStatus(strings.ToLower(c.Status))
	merged := false
	switch c.Status {
	case "NEW":
		state = forge.PRStatusOpen
	case "MERGED":
		state = forge.PRStatusMerged
		merged = true
	case "ABANDONED":
		state = forge.PRStatusClosed
	}

	head := forge.PRBranch{SHA: c.CurrentRevision}
	if rev, ok := c.Revisions[c.CurrentRevision]; ok {
		head.Ref = rev.Ref
	}
	result := forge.PullRequest{
		Number:       c.Number,
		Title:        c.Subject,
		State:        state,
		Draft:        c.WorkInProgress,
		Author:       s.convertAccount(c.Owner),
		Head:         head,
		Base:         forge.PRBranch{Ref: c.Branch},
		Merged:       merged,
		Comments:     len(c.Messages),
		Additions:    c.Insertions,
		Deletions:    c.Deletions,
		HTMLURL:      s.changeURL(c.Project, c.Number),
		CreatedAt:    parseGerritTime(c.Created),
		UpdatedAt:    parseGerritTime(c.Updated),
		ChangedFiles: 0,
	}
	if c.Submitted != "" {
		t := parseGerritTime(c.Submitted)
		result.MergedAt = &t
	}
	return result
}

func (s *gerritPRService) changeURL(project string, number int) string {
	if project == "" {
		return s.forge.baseURL + "/c/" + strconv.Itoa(number)
	}
	parts := strings.Split(project, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return s.forge.baseURL + "/c/" + strings.Join(parts, "/") + "/+/" + strconv.Itoa(number)
}

func (s *gerritPRService) Get(ctx context.Context, owner, repo string, number int) (*forge.PullRequest, error) {
	query := url.Values{}
	query.Add("o", "DETAILED_ACCOUNTS")
	query.Add("o", "CURRENT_REVISION")
	query.Add("o", "MESSAGES")

	var c gerritChangeInfo
	if err := s.forge.doJSON(ctx, http.MethodGet, "/changes/"+encodeID(strconv.Itoa(number))+"/detail", query, nil, &c); err != nil {
		return nil, err
	}
	if c.Project != projectName(owner, repo) {
		return nil, forge.ErrNotFound
	}
	result := s.convertChange(c)
	return &result, nil
}

func (s *gerritPRService) List(ctx context.Context, owner, repo string, opts forge.ListPROpts) ([]forge.PullRequest, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	start := 0
	if opts.Page > 1 {
		start = (opts.Page - 1) * perPage
	}

	var all []forge.PullRequest
	for {
		query := url.Values{}
		query.Set("q", s.query(owner, repo, opts))
		query.Set("n", strconv.Itoa(perPage))
		query.Set("S", strconv.Itoa(start))
		query.Add("o", "DETAILED_ACCOUNTS")
		query.Add("o", "CURRENT_REVISION")

		var page []gerritChangeInfo
		if err := s.forge.doJSON(ctx, http.MethodGet, "/changes/", query, nil, &page); err != nil {
			return nil, err
		}
		more := false
		for _, c := range page {
			if c.MoreChanges {
				more = true
			}
			all = append(all, s.convertChange(c))
			if opts.Limit > 0 && len(all) >= opts.Limit {
				return all[:opts.Limit], nil
			}
		}
		if !more || len(page) == 0 {
			break
		}
		start += perPage
	}
	return all, nil
}

func (s *gerritPRService) query(owner, repo string, opts forge.ListPROpts) string {
	terms := []string{"project:" + projectName(owner, repo)}
	switch opts.State {
	case "", stateOpen:
		terms = append(terms, "status:open")
	case stateMerged:
		terms = append(terms, "status:merged")
	case stateClosed:
		terms = append(terms, "status:closed")
	case stateAll:
	default:
		terms = append(terms, "status:"+opts.State)
	}
	if opts.Author != "" {
		terms = append(terms, "owner:"+opts.Author)
	}
	if opts.Base != "" {
		terms = append(terms, "branch:"+opts.Base)
	}
	return strings.Join(terms, " ")
}

func (s *gerritPRService) Create(_ context.Context, _, _ string, _ forge.CreatePROpts) (*forge.PullRequest, error) {
	return nil, forge.ErrNotSupported
}

func (s *gerritPRService) Update(_ context.Context, _, _ string, _ int, _ forge.UpdatePROpts) (*forge.PullRequest, error) {
	return nil, forge.ErrNotSupported
}

func (s *gerritPRService) Close(ctx context.Context, owner, repo string, number int) error {
	return s.forge.doJSON(ctx, http.MethodPost, "/changes/"+encodeID(strconv.Itoa(number))+"/abandon", nil, nil, nil)
}

func (s *gerritPRService) Reopen(ctx context.Context, owner, repo string, number int) error {
	return s.forge.doJSON(ctx, http.MethodPost, "/changes/"+encodeID(strconv.Itoa(number))+"/restore", nil, nil, nil)
}

func (s *gerritPRService) Merge(ctx context.Context, owner, repo string, number int, opts forge.MergePROpts) error {
	return s.forge.doJSON(ctx, http.MethodPost, "/changes/"+encodeID(strconv.Itoa(number))+"/submit", nil, nil, nil)
}

func (s *gerritPRService) Diff(ctx context.Context, owner, repo string, number int) (string, error) {
	value, err := s.forge.doText(ctx, http.MethodGet, "/changes/"+encodeID(strconv.Itoa(number))+"/revisions/current/patch", nil, nil)
	if err != nil {
		return "", err
	}
	return decodeBase64Text(value)
}

func (s *gerritPRService) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*forge.Comment, error) {
	input := map[string]string{"message": body}
	if err := s.forge.doJSON(ctx, http.MethodPost, "/changes/"+encodeID(strconv.Itoa(number))+"/revisions/current/review", nil, input, nil); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	return &forge.Comment{
		Body:      body,
		HTMLURL:   s.changeURL(projectName(owner, repo), number),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *gerritPRService) ListComments(ctx context.Context, owner, repo string, number int) ([]forge.Comment, error) {
	var messages []gerritChangeMessageInfo
	if err := s.forge.doJSON(ctx, http.MethodGet, "/changes/"+encodeID(strconv.Itoa(number))+"/messages", nil, nil, &messages); err != nil {
		return nil, err
	}
	comments := make([]forge.Comment, 0, len(messages))
	for _, msg := range messages {
		comments = append(comments, forge.Comment{
			Body:      msg.Message,
			Author:    s.convertAccount(msg.Author),
			HTMLURL:   s.changeURL(projectName(owner, repo), number),
			CreatedAt: parseGerritTime(msg.Date),
			UpdatedAt: parseGerritTime(msg.Date),
		})
	}
	return comments, nil
}

func (s *gerritPRService) ListReactions(_ context.Context, _, _ string, _ int, _ int64) ([]forge.Reaction, error) {
	return nil, forge.ErrNotSupported
}

func (s *gerritPRService) AddReaction(_ context.Context, _, _ string, _ int, _ int64, _ string) (*forge.Reaction, error) {
	return nil, forge.ErrNotSupported
}

func (s *gerritPRService) ListURL(repoHTMLURL string) string {
	if project, ok := strings.CutPrefix(repoHTMLURL, s.forge.baseURL+"/admin/repos/"); ok {
		if decoded, err := url.PathUnescape(project); err == nil {
			return s.forge.baseURL + "/q/project:" + url.QueryEscape(decoded)
		}
	}
	return strings.TrimRight(repoHTMLURL, "/") + "/+/changes"
}

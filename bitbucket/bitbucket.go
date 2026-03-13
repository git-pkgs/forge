package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	forge "github.com/git-pkgs/forge"
	"io"
	"net/http"
	"time"
)

var bitbucketAPI = "https://api.bitbucket.org/2.0"

// setBitbucketAPI overrides the Bitbucket API base URL (for testing).
func setBitbucketAPI(url string) { bitbucketAPI = url }

type bitbucketForge struct {
	token      string
	httpClient *http.Client
}

// New creates a Bitbucket forge backend.
func New(token string, hc *http.Client) forge.Forge {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &bitbucketForge{token: token, httpClient: hc}
}

type bitbucketRepoService struct {
	token      string
	httpClient *http.Client
}

func (f *bitbucketForge) Repos() forge.RepoService {
	return &bitbucketRepoService{token: f.token, httpClient: f.httpClient}
}

// Bitbucket API response types

type bbRepository struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Website     string `json:"website"`
	Language    string `json:"language"`
	IsPrivate   bool   `json:"is_private"`
	ForkPolicy  string `json:"fork_policy"`
	Size        int    `json:"size"`
	HasIssues   bool   `json:"has_issues"`
	MainBranch  *struct {
		Name string `json:"name"`
	} `json:"mainbranch"`
	Owner *struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	} `json:"owner"`
	Parent *struct {
		FullName string `json:"full_name"`
	} `json:"parent"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Avatar struct {
			Href string `json:"href"`
		} `json:"avatar"`
		Clone []struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"clone"`
	} `json:"links"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
}

type bbTagsResponse struct {
	Values []bbTag `json:"values"`
	Next   string  `json:"next"`
}

type bbTag struct {
	Name   string `json:"name"`
	Target struct {
		Hash string `json:"hash"`
	} `json:"target"`
}

type bbReposResponse struct {
	Values []bbRepository `json:"values"`
	Next   string         `json:"next"`
}

func (s *bitbucketRepoService) doJSON(ctx context.Context, method, url string, body any, v any) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return forge.ErrNotFound
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return &forge.HTTPError{StatusCode: resp.StatusCode, URL: url, Body: string(respBody)}
	}

	if v != nil {
		return json.NewDecoder(resp.Body).Decode(v)
	}
	return nil
}

func (s *bitbucketRepoService) getJSON(ctx context.Context, url string, v any) error {
	return s.doJSON(ctx, http.MethodGet, url, nil, v)
}

func convertBitbucketRepo(bb bbRepository) forge.Repository {
	result := forge.Repository{
		FullName:    bb.FullName,
		Name:        bb.Slug,
		Description: bb.Description,
		Homepage:    bb.Website,
		Language:    bb.Language,
		Private:     bb.IsPrivate,
		Size:        bb.Size,
		HasIssues:   bb.HasIssues,
		HTMLURL:     bb.Links.HTML.Href,
		LogoURL:     bb.Links.Avatar.Href,
	}

	for _, c := range bb.Links.Clone {
		switch c.Name {
		case "https":
			result.CloneURL = c.Href
		case "ssh":
			result.SSHURL = c.Href
		}
	}

	if bb.Owner != nil {
		result.Owner = bb.Owner.Username
	}

	if bb.MainBranch != nil {
		result.DefaultBranch = bb.MainBranch.Name
	}

	if bb.Parent != nil {
		result.Fork = true
		result.SourceName = bb.Parent.FullName
	}

	if t, err := time.Parse(time.RFC3339, bb.CreatedOn); err == nil {
		result.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, bb.UpdatedOn); err == nil {
		result.UpdatedAt = t
	}

	return result
}

func (s *bitbucketRepoService) Get(ctx context.Context, owner, repo string) (*forge.Repository, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s", bitbucketAPI, owner, repo)
	var bb bbRepository
	if err := s.getJSON(ctx, url, &bb); err != nil {
		return nil, err
	}

	result := convertBitbucketRepo(bb)
	return &result, nil
}

func (s *bitbucketRepoService) List(ctx context.Context, owner string, opts forge.ListRepoOpts) ([]forge.Repository, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 100
	}

	var all []forge.Repository
	url := fmt.Sprintf("%s/repositories/%s?pagelen=%d", bitbucketAPI, owner, perPage)

	for url != "" {
		var page bbReposResponse
		if err := s.getJSON(ctx, url, &page); err != nil {
			if errors.Is(err, forge.ErrNotFound) {
				return nil, forge.ErrOwnerNotFound
			}
			return nil, err
		}
		for _, bb := range page.Values {
			all = append(all, convertBitbucketRepo(bb))
		}
		url = page.Next
	}

	return forge.FilterRepos(all, opts), nil
}

func (s *bitbucketRepoService) Create(ctx context.Context, opts forge.CreateRepoOpts) (*forge.Repository, error) {
	owner := opts.Owner
	if owner == "" {
		return nil, fmt.Errorf("bitbucket: owner is required for repo creation")
	}

	body := map[string]any{
		"scm": "git",
	}
	if opts.Description != "" {
		body["description"] = opts.Description
	}
	if opts.Visibility == forge.VisibilityPrivate {
		body["is_private"] = true
	} else {
		body["is_private"] = false
	}
	url := fmt.Sprintf("%s/repositories/%s/%s", bitbucketAPI, owner, opts.Name)
	var bb bbRepository
	if err := s.doJSON(ctx, http.MethodPost, url, body, &bb); err != nil {
		return nil, err
	}

	result := convertBitbucketRepo(bb)
	return &result, nil
}

func (s *bitbucketRepoService) Edit(ctx context.Context, owner, repo string, opts forge.EditRepoOpts) (*forge.Repository, error) {
	body := map[string]any{}

	if opts.Description != nil {
		body["description"] = *opts.Description
	}
	if opts.Homepage != nil {
		body["website"] = *opts.Homepage
	}
	if opts.HasIssues != nil {
		body["has_issues"] = *opts.HasIssues
	}

	switch opts.Visibility {
	case forge.VisibilityPrivate:
		body["is_private"] = true
	case forge.VisibilityPublic:
		body["is_private"] = false
	}

	if len(body) == 0 {
		return s.Get(ctx, owner, repo)
	}

	url := fmt.Sprintf("%s/repositories/%s/%s", bitbucketAPI, owner, repo)
	var bb bbRepository
	if err := s.doJSON(ctx, http.MethodPut, url, body, &bb); err != nil {
		return nil, err
	}

	result := convertBitbucketRepo(bb)
	return &result, nil
}

func (s *bitbucketRepoService) Delete(ctx context.Context, owner, repo string) error {
	url := fmt.Sprintf("%s/repositories/%s/%s", bitbucketAPI, owner, repo)
	return s.doJSON(ctx, http.MethodDelete, url, nil, nil)
}

func (s *bitbucketRepoService) Fork(ctx context.Context, owner, repo string, opts forge.ForkRepoOpts) (*forge.Repository, error) {
	body := map[string]any{}
	if opts.Name != "" {
		body["name"] = opts.Name
	}
	if opts.Owner != "" {
		body["workspace"] = map[string]string{"slug": opts.Owner}
	}

	url := fmt.Sprintf("%s/repositories/%s/%s/forks", bitbucketAPI, owner, repo)
	var bb bbRepository
	if err := s.doJSON(ctx, http.MethodPost, url, body, &bb); err != nil {
		return nil, err
	}

	result := convertBitbucketRepo(bb)
	return &result, nil
}

func (s *bitbucketRepoService) ListForks(ctx context.Context, owner, repo string, opts forge.ListForksOpts) ([]forge.Repository, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 100
	}

	var all []forge.Repository
	url := fmt.Sprintf("%s/repositories/%s/%s/forks?pagelen=%d", bitbucketAPI, owner, repo, perPage)

	for url != "" {
		var page bbReposResponse
		if err := s.getJSON(ctx, url, &page); err != nil {
			return nil, err
		}
		for _, bb := range page.Values {
			all = append(all, convertBitbucketRepo(bb))
		}
		if opts.Limit > 0 && len(all) >= opts.Limit {
			break
		}
		url = page.Next
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *bitbucketRepoService) ListTags(ctx context.Context, owner, repo string) ([]forge.Tag, error) {
	var allTags []forge.Tag
	url := fmt.Sprintf("%s/repositories/%s/%s/refs/tags?pagelen=100", bitbucketAPI, owner, repo)

	for url != "" {
		var page bbTagsResponse
		if err := s.getJSON(ctx, url, &page); err != nil {
			return nil, err
		}
		for _, t := range page.Values {
			allTags = append(allTags, forge.Tag{
				Name:   t.Name,
				Commit: t.Target.Hash,
			})
		}
		url = page.Next
	}
	return allTags, nil
}

func (s *bitbucketRepoService) Search(ctx context.Context, opts forge.SearchRepoOpts) ([]forge.Repository, error) {
	// Bitbucket doesn't have a global repo search API.
	// The closest is searching within a workspace, which requires an owner.
	return nil, forge.ErrNotSupported
}

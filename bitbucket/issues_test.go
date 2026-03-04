package bitbucket

import (
	"context"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBitbucketGetIssue(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/issues/1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bbIssue{
			ID:    1,
			Title: "Test issue",
			Content: struct {
				Raw string `json:"raw"`
			}{Raw: "Issue body"},
			State: "open",
			Reporter: &struct {
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
			}{Username: "reporter1"},
			Links: struct {
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
			}{HTML: struct {
				Href string `json:"href"`
			}{Href: "https://bitbucket.org/atlassian/stash/issues/1"}},
			CreatedOn: "2024-01-01T00:00:00Z",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	s := &bitbucketIssueService{httpClient: srv.Client()}
	issue, err := s.Get(context.Background(), "atlassian", "stash", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqualInt(t, "Number", 1, issue.Number)
	assertEqual(t, "Title", "Test issue", issue.Title)
	assertEqual(t, "Body", "Issue body", issue.Body)
	assertEqual(t, "State", "open", issue.State)
	assertEqual(t, "Author.Login", "reporter1", issue.Author.Login)
}

func TestBitbucketListIssues(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/issues", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bbIssuesResponse{
			Values: []bbIssue{
				{ID: 1, Title: "First", State: "open"},
				{ID: 2, Title: "Second", State: "new"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	s := &bitbucketIssueService{httpClient: srv.Client()}
	issues, err := s.List(context.Background(), "atlassian", "stash", forge.ListIssueOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	assertEqual(t, "issues[0].Title", "First", issues[0].Title)
	assertEqual(t, "issues[0].State", "open", issues[0].State)
	assertEqual(t, "issues[1].Title", "Second", issues[1].Title)
	assertEqual(t, "issues[1].State", "open", issues[1].State) // "new" maps to "open"
}

func TestBitbucketCreateIssue(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /2.0/repositories/atlassian/stash/issues", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(bbIssue{
			ID:    5,
			Title: body["title"].(string),
			State: "new",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	s := &bitbucketIssueService{httpClient: srv.Client()}
	issue, err := s.Create(context.Background(), "atlassian", "stash", forge.CreateIssueOpts{
		Title: "New issue",
		Body:  "Details",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualInt(t, "Number", 5, issue.Number)
	assertEqual(t, "Title", "New issue", issue.Title)
}

func TestBitbucketCloseIssue(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /2.0/repositories/atlassian/stash/issues/1", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["state"] != "resolved" {
			t.Errorf("expected state=resolved, got %v", body["state"])
		}
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	s := &bitbucketIssueService{httpClient: srv.Client()}
	if err := s.Close(context.Background(), "atlassian", "stash", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBitbucketCreateComment(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /2.0/repositories/atlassian/stash/issues/1/comments", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(bbComment{
			ID: 10,
			Content: struct {
				Raw string `json:"raw"`
			}{Raw: "A comment"},
			User: &struct {
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
			}{Username: "commenter"},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	s := &bitbucketIssueService{httpClient: srv.Client()}
	comment, err := s.CreateComment(context.Background(), "atlassian", "stash", 1, "A comment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comment.ID != 10 {
		t.Errorf("expected ID=10, got %d", comment.ID)
	}
	assertEqual(t, "Body", "A comment", comment.Body)
	assertEqual(t, "Author.Login", "commenter", comment.Author.Login)
}

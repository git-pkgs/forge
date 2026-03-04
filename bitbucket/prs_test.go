package bitbucket

import (
	"context"
	"encoding/json"
	forge "github.com/git-pkgs/forge"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBitbucketGetPR(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/pullrequests/1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bbPullRequest{
			ID:          1,
			Title:       "Add feature",
			Description: "New feature PR",
			State:       "OPEN",
			Source: struct {
				Branch struct {
					Name string `json:"name"`
				} `json:"branch"`
			}{Branch: struct {
				Name string `json:"name"`
			}{Name: "feature-branch"}},
			Destination: struct {
				Branch struct {
					Name string `json:"name"`
				} `json:"branch"`
			}{Branch: struct {
				Name string `json:"name"`
			}{Name: "main"}},
			Author: &struct {
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
			}{Username: "author1"},
			Reviewers: []struct {
				Username    string `json:"username"`
				DisplayName string `json:"display_name"`
			}{
				{Username: "reviewer1"},
			},
			CommentCount: 3,
			CreatedOn:    "2024-01-01T00:00:00+00:00",
			Links: struct {
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
				Diff struct {
					Href string `json:"href"`
				} `json:"diff"`
			}{
				HTML: struct {
					Href string `json:"href"`
				}{Href: "https://bitbucket.org/atlassian/stash/pull-requests/1"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	s := &bitbucketPRService{httpClient: srv.Client()}
	pr, err := s.Get(context.Background(), "atlassian", "stash", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqualInt(t, "Number", 1, pr.Number)
	assertEqual(t, "Title", "Add feature", pr.Title)
	assertEqual(t, "Body", "New feature PR", pr.Body)
	assertEqual(t, "State", "open", pr.State)
	assertEqual(t, "Head", "feature-branch", pr.Head)
	assertEqual(t, "Base", "main", pr.Base)
	assertEqual(t, "Author.Login", "author1", pr.Author.Login)
	assertEqualInt(t, "Comments", 3, pr.Comments)
	assertEqualBool(t, "Merged", false, pr.Merged)

	if len(pr.Reviewers) != 1 {
		t.Fatalf("expected 1 reviewer, got %d", len(pr.Reviewers))
	}
	assertEqual(t, "Reviewers[0].Login", "reviewer1", pr.Reviewers[0].Login)
}

func TestBitbucketGetPRMerged(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/pullrequests/2", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bbPullRequest{
			ID:    2,
			Title: "Merged PR",
			State: "MERGED",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	s := &bitbucketPRService{httpClient: srv.Client()}
	pr, err := s.Get(context.Background(), "atlassian", "stash", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "State", "merged", pr.State)
	assertEqualBool(t, "Merged", true, pr.Merged)
}

func TestBitbucketListPRs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/pullrequests", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bbPRsResponse{
			Values: []bbPullRequest{
				{ID: 1, Title: "First", State: "OPEN"},
				{ID: 2, Title: "Second", State: "OPEN"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	s := &bitbucketPRService{httpClient: srv.Client()}
	prs, err := s.List(context.Background(), "atlassian", "stash", forge.ListPROpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}
	assertEqual(t, "prs[0].Title", "First", prs[0].Title)
	assertEqual(t, "prs[1].Title", "Second", prs[1].Title)
}

func TestBitbucketCreatePR(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /2.0/repositories/atlassian/stash/pullrequests", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(bbPullRequest{
			ID:    5,
			Title: body["title"].(string),
			State: "OPEN",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	s := &bitbucketPRService{httpClient: srv.Client()}
	pr, err := s.Create(context.Background(), "atlassian", "stash", forge.CreatePROpts{
		Title: "New PR",
		Body:  "Description",
		Head:  "feature",
		Base:  "main",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualInt(t, "Number", 5, pr.Number)
	assertEqual(t, "Title", "New PR", pr.Title)
}

func TestBitbucketClosePR(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /2.0/repositories/atlassian/stash/pullrequests/1/decline", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	s := &bitbucketPRService{httpClient: srv.Client()}
	if err := s.Close(context.Background(), "atlassian", "stash", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBitbucketMergePR(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /2.0/repositories/atlassian/stash/pullrequests/1/merge", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	s := &bitbucketPRService{httpClient: srv.Client()}
	if err := s.Merge(context.Background(), "atlassian", "stash", 1, forge.MergePROpts{Method: "merge_commit"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBitbucketDiffPR(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/pullrequests/1/diff", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("diff --git a/file.txt b/file.txt\n"))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	s := &bitbucketPRService{httpClient: srv.Client()}
	diff, err := s.Diff(context.Background(), "atlassian", "stash", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff != "diff --git a/file.txt b/file.txt\n" {
		t.Errorf("unexpected diff: %q", diff)
	}
}

func TestBitbucketPRCreateComment(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /2.0/repositories/atlassian/stash/pullrequests/1/comments", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(bbComment{
			ID: 50,
			Content: struct {
				Raw string `json:"raw"`
			}{Raw: "Looks good"},
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
			}{Username: "reviewer"},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	s := &bitbucketPRService{httpClient: srv.Client()}
	comment, err := s.CreateComment(context.Background(), "atlassian", "stash", 1, "Looks good")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comment.ID != 50 {
		t.Errorf("expected ID=50, got %d", comment.ID)
	}
	assertEqual(t, "Body", "Looks good", comment.Body)
	assertEqual(t, "Author.Login", "reviewer", comment.Author.Login)
}

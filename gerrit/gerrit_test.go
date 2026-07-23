package gerrit

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	forge "github.com/git-pkgs/forge"
)

const xssi = ")]}'\n"

func newTestForge(handler http.Handler) (*gerritForge, func()) {
	srv := httptest.NewServer(handler)
	f := New(srv.URL, "", srv.Client()).(*gerritForge)
	return f, srv.Close
}

func TestRepoGetStripsXSSIAndMapsProject(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /projects/org%2Frepo", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, xssi+`{
			"id":"org%2Frepo",
			"name":"org/repo",
			"description":"demo",
			"state":"ACTIVE"
		}`)
	})
	mux.HandleFunc("GET /projects/org%2Frepo/HEAD", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, xssi+`"refs/heads/main"`)
	})
	f, done := newTestForge(mux)
	defer done()

	repo, err := f.Repos().Get(context.Background(), "org", "repo")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if repo.FullName != "org/repo" || repo.Owner != "org" || repo.Name != "repo" {
		t.Fatalf("unexpected repo identity: %+v", repo)
	}
	if repo.Description != "demo" {
		t.Fatalf("Description = %q, want demo", repo.Description)
	}
	if repo.DefaultBranch != "main" {
		t.Fatalf("DefaultBranch = %q, want main", repo.DefaultBranch)
	}
	if repo.HasIssues {
		t.Fatalf("Gerrit repo should not advertise issues")
	}
	if !repo.PullRequestsEnabled {
		t.Fatalf("Gerrit changes should map to pull requests")
	}
}

func TestRepoListUsesProjectPrefixAndLimit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /projects/", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("p"); got != "team/" {
			t.Fatalf("prefix query = %q, want team/", got)
		}
		if got := r.URL.Query().Get("n"); got != "2" {
			t.Fatalf("page size = %q, want 2", got)
		}
		_, _ = fmt.Fprint(w, xssi+`{
			"team/a":{"id":"team%2Fa","description":"A"},
			"team/b":{"id":"team%2Fb","description":"B","_more_projects":true}
		}`)
	})
	f, done := newTestForge(mux)
	defer done()

	repos, err := f.Repos().List(context.Background(), "team", forge.ListRepoOpts{PerPage: 2, Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(repos) != 2 || repos[0].FullName != "team/a" || repos[1].FullName != "team/b" {
		t.Fatalf("unexpected repos: %+v", repos)
	}
}

func TestBranchesAndTags(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /projects/org%2Frepo/branches/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, xssi+`[
			{"ref":"refs/heads/main","revision":"abc"},
			{"ref":"refs/heads/dev","revision":"def"}
		]`)
	})
	mux.HandleFunc("GET /projects/org%2Frepo/tags/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, xssi+`[
			{"ref":"refs/tags/v1.0.0","revision":"111"}
		]`)
	})
	f, done := newTestForge(mux)
	defer done()

	branches, err := f.Branches().List(context.Background(), "org", "repo", forge.ListBranchOpts{Limit: 1})
	if err != nil {
		t.Fatalf("branches List: %v", err)
	}
	if len(branches) != 1 || branches[0].Name != "main" || branches[0].SHA != "abc" {
		t.Fatalf("unexpected branches: %+v", branches)
	}

	tags, err := f.Repos().ListTags(context.Background(), "org", "repo")
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "v1.0.0" || tags[0].Commit != "111" {
		t.Fatalf("unexpected tags: %+v", tags)
	}
}

func TestFileGetDecodesBase64(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /projects/org%2Frepo/branches/main/files/README.md/content", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, base64.StdEncoding.EncodeToString([]byte("hello\n")))
	})
	f, done := newTestForge(mux)
	defer done()

	file, err := f.Files().Get(context.Background(), "org", "repo", "README.md", "main")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(file.Content) != "hello\n" {
		t.Fatalf("content = %q, want hello newline", string(file.Content))
	}
}

func TestPullRequestGetAndDiff(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /changes/42/detail", func(w http.ResponseWriter, r *http.Request) {
		options := r.URL.Query()["o"]
		if len(options) == 0 {
			t.Fatalf("expected Gerrit option query values")
		}
		_, _ = fmt.Fprint(w, xssi+`{
			"id":"org%2Frepo~main~Iabc",
			"project":"org/repo",
			"branch":"main",
			"change_id":"Iabc",
			"subject":"Add Gerrit",
			"status":"MERGED",
			"owner":{"_account_id":7,"username":"alice","name":"Alice"},
			"created":"2026-03-10 16:09:43.000000000",
			"updated":"2026-03-11 16:09:43.000000000",
			"submitted":"2026-03-12 16:09:43.000000000",
			"_number":42,
			"insertions":10,
			"deletions":2,
			"current_revision":"deadbeef",
			"revisions":{"deadbeef":{"_number":1,"ref":"refs/changes/42/42/1"}},
			"messages":[{"id":"m1","message":"Uploaded patch set 1"}]
		}`)
	})
	mux.HandleFunc("GET /changes/42/revisions/current/patch", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, base64.StdEncoding.EncodeToString([]byte("diff --git a/a b/a\n")))
	})
	f, done := newTestForge(mux)
	defer done()

	pr, err := f.PullRequests().Get(context.Background(), "org", "repo", 42)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if pr.State != "merged" || !pr.Merged || pr.Number != 42 {
		t.Fatalf("unexpected PR state: %+v", pr)
	}
	if pr.Author.Login != "alice" {
		t.Fatalf("author = %+v, want alice", pr.Author)
	}
	if pr.Head.SHA != "deadbeef" || pr.Head.Ref != "refs/changes/42/42/1" {
		t.Fatalf("head = %+v", pr.Head)
	}

	diff, err := f.PullRequests().Diff(context.Background(), "org", "repo", 42)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if diff != "diff --git a/a b/a\n" {
		t.Fatalf("diff = %q", diff)
	}
}

func TestPullRequestGetRejectsDifferentProject(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /changes/42/detail", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, xssi+`{
			"project":"other/repo",
			"branch":"main",
			"subject":"Wrong project",
			"status":"NEW",
			"_number":42
		}`)
	})
	f, done := newTestForge(mux)
	defer done()

	_, err := f.PullRequests().Get(context.Background(), "org", "repo", 42)
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("Get error = %v, want ErrNotFound", err)
	}
}

func TestPullRequestListBuildsGerritQuery(t *testing.T) {
	mux := http.NewServeMux()
	requests := 0
	mux.HandleFunc("GET /changes/", func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests > 1 {
			_, _ = fmt.Fprint(w, xssi+`[]`)
			return
		}
		got, _ := url.QueryUnescape(r.URL.Query().Get("q"))
		want := "project:org/repo status:open owner:alice branch:main"
		if got != want {
			t.Fatalf("query = %q, want %q", got, want)
		}
		_, _ = fmt.Fprint(w, xssi+`[
			{"project":"org/repo","branch":"main","subject":"One","status":"NEW","_number":1,"_more_changes":true}
		]`)
	})
	f, done := newTestForge(mux)
	defer done()

	prs, err := f.PullRequests().List(context.Background(), "org", "repo", forge.ListPROpts{
		State:  "open",
		Author: "alice",
		Base:   "main",
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(prs) != 1 || prs[0].State != "open" {
		t.Fatalf("unexpected PRs: %+v", prs)
	}
}

func TestUnsupportedServicesReturnErrNotSupported(t *testing.T) {
	f, done := newTestForge(http.NewServeMux())
	defer done()

	_, err := f.Issues().List(context.Background(), "org", "repo", forge.ListIssueOpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Fatalf("Issues().List error = %v, want ErrNotSupported", err)
	}

	_, err = f.PullRequests().Create(context.Background(), "org", "repo", forge.CreatePROpts{})
	if !errors.Is(err, forge.ErrNotSupported) {
		t.Fatalf("PullRequests().Create error = %v, want ErrNotSupported", err)
	}
}

func TestParsePath(t *testing.T) {
	f := New("https://gerrit.example.com", "", nil)
	ref, err := f.ParsePath([]string{"c", "plugins", "replication", "+", "123"})
	if err != nil {
		t.Fatalf("ParsePath: %v", err)
	}
	if ref.Owner != "plugins" || ref.Repo != "replication" || ref.Type != forge.ResourceTypePR || ref.Number != 123 {
		t.Fatalf("unexpected ref: %+v", ref)
	}
}

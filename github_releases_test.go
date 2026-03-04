package forges

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v82/github"
)

func newTestGitHubReleaseService(srv *httptest.Server) *gitHubReleaseService {
	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	return &gitHubReleaseService{client: c}
}

func TestGitHubReleaseList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
			{
				"tag_name": "v1.0.0",
				"name": "Release 1.0.0",
				"body": "First release",
				"draft": false,
				"prerelease": false,
				"html_url": "https://github.com/octocat/hello/releases/tag/v1.0.0",
				"author": {"login": "octocat", "avatar_url": "https://avatars.githubusercontent.com/u/1?v=4"},
				"assets": [
					{
						"id": 1,
						"name": "hello.tar.gz",
						"size": 1024,
						"download_count": 42,
						"browser_download_url": "https://github.com/octocat/hello/releases/download/v1.0.0/hello.tar.gz"
					}
				]
			}
		]`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubReleaseService(srv)
	releases, err := svc.List(context.Background(), "octocat", "hello", ListReleaseOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("expected 1 release, got %d", len(releases))
	}

	r := releases[0]
	assertEqual(t, "TagName", "v1.0.0", r.TagName)
	assertEqual(t, "Title", "Release 1.0.0", r.Title)
	assertEqual(t, "Body", "First release", r.Body)
	assertEqualBool(t, "Draft", false, r.Draft)
	assertEqualBool(t, "Prerelease", false, r.Prerelease)
	assertEqual(t, "Author.Login", "octocat", r.Author.Login)
	assertEqualInt(t, "len(Assets)", 1, len(r.Assets))
	assertEqual(t, "Assets[0].Name", "hello.tar.gz", r.Assets[0].Name)
	assertEqualInt(t, "Assets[0].Size", 1024, r.Assets[0].Size)
	assertEqualInt(t, "Assets[0].DownloadCount", 42, r.Assets[0].DownloadCount)
}

func TestGitHubReleaseGet(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"tag_name": "v1.0.0",
			"name": "Release 1.0.0",
			"body": "First release",
			"draft": false,
			"prerelease": false,
			"html_url": "https://github.com/octocat/hello/releases/tag/v1.0.0"
		}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubReleaseService(srv)
	r, err := svc.Get(context.Background(), "octocat", "hello", "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "TagName", "v1.0.0", r.TagName)
	assertEqual(t, "Title", "Release 1.0.0", r.Title)
}

func TestGitHubReleaseGetNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/tags/v999.0.0", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message": "Not Found"}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubReleaseService(srv)
	_, err := svc.Get(context.Background(), "octocat", "hello", "v999.0.0")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGitHubReleaseGetLatest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"tag_name": "v2.0.0",
			"name": "Release 2.0.0",
			"draft": false,
			"prerelease": false
		}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubReleaseService(srv)
	r, err := svc.GetLatest(context.Background(), "octocat", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "TagName", "v2.0.0", r.TagName)
}

func TestGitHubReleaseCreate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/repos/octocat/hello/releases", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"tag_name": "v3.0.0",
			"name": "Release 3.0.0",
			"body": "New release",
			"draft": true,
			"prerelease": false,
			"html_url": "https://github.com/octocat/hello/releases/tag/v3.0.0"
		}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubReleaseService(srv)
	r, err := svc.Create(context.Background(), "octocat", "hello", CreateReleaseOpts{
		TagName: "v3.0.0",
		Title:   "Release 3.0.0",
		Body:    "New release",
		Draft:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "TagName", "v3.0.0", r.TagName)
	assertEqualBool(t, "Draft", true, r.Draft)
}

func TestGitHubReleaseDelete(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello/releases/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id": 1, "tag_name": "v1.0.0"}`)
	})
	mux.HandleFunc("DELETE /api/v3/repos/octocat/hello/releases/1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := newTestGitHubReleaseService(srv)
	err := svc.Delete(context.Background(), "octocat", "hello", "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

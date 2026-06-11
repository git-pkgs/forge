package tangled

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	forges "github.com/git-pkgs/forge"
)

func TestRepoGetUsesAppviewMetadataAndBranches(t *testing.T) {
	srv := tangledTestServer(t)
	f := New(srv.URL, "", srv.Client())

	repo, err := f.Repos().Get(context.Background(), "tangled.org", "core")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if repo.FullName != "tangled.org/core" {
		t.Errorf("FullName = %q", repo.FullName)
	}
	if repo.CloneURL != srv.URL+"/tangled.org/core" {
		t.Errorf("CloneURL = %q", repo.CloneURL)
	}
	if repo.Description != "Tangled core app" {
		t.Errorf("Description = %q", repo.Description)
	}
	if repo.DefaultBranch != "master" {
		t.Errorf("DefaultBranch = %q", repo.DefaultBranch)
	}
}

func TestBranchListUsesTangledXRPC(t *testing.T) {
	srv := tangledTestServer(t)
	f := New(srv.URL, "", srv.Client())

	branches, err := f.Branches().List(context.Background(), "tangled.org", "core", forges.ListBranchOpts{Limit: 1})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(branches) != 1 {
		t.Fatalf("len(branches) = %d", len(branches))
	}
	if branches[0].Name != "master" || branches[0].SHA != "abc123" || !branches[0].Default {
		t.Errorf("unexpected branch: %+v", branches[0])
	}
}

func TestRepoListTagsUsesTangledXRPC(t *testing.T) {
	srv := tangledTestServer(t)
	f := New(srv.URL, "", srv.Client())

	tags, err := f.Repos().ListTags(context.Background(), "tangled.org", "core")
	if err != nil {
		t.Fatalf("ListTags returned error: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("len(tags) = %d", len(tags))
	}
	if tags[0].Name != "v0.1.0" || tags[0].Commit != "def456" {
		t.Errorf("unexpected tag: %+v", tags[0])
	}
}

func TestFileListUsesTangledTreeXRPC(t *testing.T) {
	srv := tangledTestServer(t)
	f := New(srv.URL, "", srv.Client())

	entries, err := f.Files().List(context.Background(), "tangled.org", "core", "api", "master")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d", len(entries))
	}
	if entries[0].Path != "api/tangled" || entries[0].Type != "dir" {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].Path != "api/defs.go" || entries[1].Type != "file" || entries[1].Size != 1234 {
		t.Errorf("unexpected second entry: %+v", entries[1])
	}
}

func TestParsePath(t *testing.T) {
	f := New("https://tangled.org", "", nil)
	ref, err := f.ParsePath([]string{"did:plc:abc", "core", "issues", "42"})
	if err != nil {
		t.Fatalf("ParsePath returned error: %v", err)
	}
	if ref.Owner != "did:plc:abc" || ref.Repo != "core" || ref.Type != forges.ResourceTypeIssue || ref.Number != 42 {
		t.Errorf("unexpected ref: %+v", ref)
	}
}

func tangledTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /tangled.org/core", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `<html><head>
<meta name="vcs:clone" content="%s/tangled.org/core">
<meta name="description" content="Tangled core app">
</head><body data-star-subject-at="at://did:plc:owner/sh.tangled.repo/core"></body></html>`, "http://"+r.Host)
	})
	mux.HandleFunc("GET /xrpc/sh.tangled.git.temp.listBranches", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("repo"); got != "did:plc:owner" {
			t.Errorf("repo query = %q", got)
		}
		_, _ = fmt.Fprint(w, `{"branches":[{"name":"refs/heads/master","target":{"hash":"abc123"},"default":true},{"name":"next","sha":"999"}]}`)
	})
	mux.HandleFunc("GET /xrpc/sh.tangled.git.temp.listTags", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("repo"); got != "did:plc:owner" {
			t.Errorf("repo query = %q", got)
		}
		_, _ = fmt.Fprint(w, `{"tags":[{"name":"refs/tags/v0.1.0","target":{"hash":"def456"}}]}`)
	})
	mux.HandleFunc("GET /xrpc/sh.tangled.git.temp.getTree", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("path"); got != "api" {
			t.Errorf("path query = %q", got)
		}
		if got := r.URL.Query().Get("ref"); got != "master" {
			t.Errorf("ref query = %q", got)
		}
		_, _ = fmt.Fprint(w, `{"tree":[{"name":"tangled","mode":"040000"},{"name":"defs.go","mode":"100644","size":1234}]}`)
	})
	return httptest.NewServer(mux)
}

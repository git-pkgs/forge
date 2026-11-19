package bitbucket

import (
	"context"
	"errors"
	"testing"

	forge "github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/httpreplay"
)

// These tests replay responses recorded from api.bitbucket.org. Re-record
// them with FORGE_RECORD=1 go test ./bitbucket -run TestReplay.

func TestReplayGetRepo(t *testing.T) {
	f := New("", httpreplay.Client(t))
	repo, err := f.Repos().Get(context.Background(), "tutorials", "markdowndemo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "FullName", "tutorials/markdowndemo", repo.FullName)
	assertEqual(t, "Owner", "tutorials-team", repo.Owner)
	assertEqual(t, "Name", "markdowndemo", repo.Name)
	assertEqual(t, "HTMLURL", "https://bitbucket.org/tutorials/markdowndemo", repo.HTMLURL)
	assertEqual(t, "DefaultBranch", "master", repo.DefaultBranch)
	assertEqualBool(t, "Private", false, repo.Private)
	if repo.CloneURL == "" {
		t.Error("CloneURL is empty")
	}
}

func TestReplayGetRepoNotFound(t *testing.T) {
	f := New("", httpreplay.Client(t))
	_, err := f.Repos().Get(context.Background(), "tutorials", "this-repository-does-not-exist")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestReplayGetFile(t *testing.T) {
	f := New("", httpreplay.Client(t))
	file, err := f.Files().Get(context.Background(), "atlassian", "atlassian-connect-express", ".nvmrc", "v12.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Name", ".nvmrc", file.Name)
	assertEqual(t, "Path", ".nvmrc", file.Path)
	assertEqual(t, "Content", "18.20.7", string(file.Content))
}

func TestReplayListFiles(t *testing.T) {
	f := New("", httpreplay.Client(t))
	entries, err := f.Files().List(context.Background(), "atlassian", "atlassian-connect-express", "lib", "v12.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	types := map[string]string{}
	for _, e := range entries {
		types[e.Path] = e.Type
	}
	assertEqual(t, "lib/internal", "dir", types["lib/internal"])
	assertEqual(t, "lib/index.js", "file", types["lib/index.js"])
}

func TestReplayResolveCommit(t *testing.T) {
	f := New("", httpreplay.Client(t))
	sha, err := f.Commits().ResolveCommit(context.Background(), "atlassian", "atlassian-connect-express", "v12.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "SHA", "fd61ca641c0db3ab42ad10bc1b2b7190033aee77", sha)
}

func TestReplayResolveCommitSlashedBranch(t *testing.T) {
	f := New("", httpreplay.Client(t))
	sha, err := f.Commits().ResolveCommit(context.Background(), "atlassian", "atlassian-connect-express", "inno/forge-remote-support")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "SHA", "cd987445da673a61046669f8d42e05b50255007d", sha)
}

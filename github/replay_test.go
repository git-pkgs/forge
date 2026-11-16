package github

import (
	"context"
	"errors"
	"testing"

	forge "github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/httpreplay"
)

// These tests replay responses recorded from api.github.com. Re-record them
// with FORGE_RECORD=1 go test ./github -run TestReplay.

func TestReplayGetRepo(t *testing.T) {
	f := New("", httpreplay.Client(t))
	repo, err := f.Repos().Get(context.Background(), "octocat", "Hello-World")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "FullName", "octocat/Hello-World", repo.FullName)
	assertEqual(t, "Owner", "octocat", repo.Owner)
	assertEqual(t, "Name", "Hello-World", repo.Name)
	assertEqual(t, "HTMLURL", "https://github.com/octocat/Hello-World", repo.HTMLURL)
	assertEqual(t, "CloneURL", "https://github.com/octocat/Hello-World.git", repo.CloneURL)
	assertEqual(t, "SSHURL", "git@github.com:octocat/Hello-World.git", repo.SSHURL)
	assertEqual(t, "DefaultBranch", "master", repo.DefaultBranch)
	assertEqualBool(t, "Fork", false, repo.Fork)
	assertEqualBool(t, "Private", false, repo.Private)
	if repo.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestReplayGetRepoNotFound(t *testing.T) {
	f := New("", httpreplay.Client(t))
	_, err := f.Repos().Get(context.Background(), "octocat", "this-repository-does-not-exist")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestReplayListBranches(t *testing.T) {
	f := New("", httpreplay.Client(t))
	branches, err := f.Branches().List(context.Background(), "octocat", "Hello-World", forge.ListBranchOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	byName := make(map[string]forge.Branch, len(branches))
	for _, b := range branches {
		byName[b.Name] = b
	}
	master, ok := byName["master"]
	if !ok {
		t.Fatalf("master branch missing from %v", branches)
	}
	assertEqual(t, "master.SHA", "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d", master.SHA)
}

func TestReplayGetFile(t *testing.T) {
	f := New("", httpreplay.Client(t))
	file, err := f.Files().Get(context.Background(), "octocat", "Hello-World", "README", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Name", "README", file.Name)
	assertEqual(t, "Path", "README", file.Path)
	assertEqual(t, "Content", "Hello World!\n", string(file.Content))
}

func TestReplayResolveCommit(t *testing.T) {
	f := New("", httpreplay.Client(t))
	sha, err := f.Commits().ResolveCommit(context.Background(), "octocat", "Hello-World", "7fd1a60")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "SHA", "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d", sha)
}

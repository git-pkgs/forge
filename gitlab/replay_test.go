//go:build !tinygo

package gitlab

import (
	"context"
	"errors"
	"testing"

	forge "github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/httpreplay"
)

// These tests replay responses recorded from gitlab.com. Re-record them with
// FORGE_RECORD=1 go test ./gitlab -run TestReplay.

func TestReplayGetRepo(t *testing.T) {
	f := New("https://gitlab.com", "", httpreplay.Client(t))
	repo, err := f.Repos().Get(context.Background(), "gitlab-org", "gitlab-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "FullName", "gitlab-org/gitlab-test", repo.FullName)
	assertEqual(t, "Owner", "gitlab-org", repo.Owner)
	assertEqual(t, "Name", "gitlab-test", repo.Name)
	assertEqual(t, "HTMLURL", "https://gitlab.com/gitlab-org/gitlab-test", repo.HTMLURL)
	assertEqual(t, "CloneURL", "https://gitlab.com/gitlab-org/gitlab-test.git", repo.CloneURL)
	assertEqual(t, "SSHURL", "git@gitlab.com:gitlab-org/gitlab-test.git", repo.SSHURL)
	assertEqual(t, "DefaultBranch", "master", repo.DefaultBranch)
	assertEqualBool(t, "Private", false, repo.Private)
	if repo.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestReplayGetRepoNotFound(t *testing.T) {
	f := New("https://gitlab.com", "", httpreplay.Client(t))
	_, err := f.Repos().Get(context.Background(), "gitlab-org", "this-project-does-not-exist")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestReplayListBranches(t *testing.T) {
	f := New("https://gitlab.com", "", httpreplay.Client(t))
	branches, err := f.Branches().List(context.Background(), "gitlab-org", "gitlab-test", forge.ListBranchOpts{Limit: 3, PerPage: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Two pages of two branches, capped at the limit.
	assertEqualInt(t, "len(branches)", 3, len(branches))
	for _, b := range branches {
		if b.Name == "" || len(b.SHA) != 40 {
			t.Errorf("branch missing name or SHA: %+v", b)
		}
	}
}

func TestReplayGetFile(t *testing.T) {
	f := New("https://gitlab.com", "", httpreplay.Client(t))
	file, err := f.Files().Get(context.Background(), "gitlab-org", "gitlab-test", "VERSION", "master")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Name", "VERSION", file.Name)
	assertEqual(t, "Path", "VERSION", file.Path)
	assertEqual(t, "Content", "6.7.0.pre\n", string(file.Content))
}

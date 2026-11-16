package gitea

import (
	"context"
	"errors"
	"strings"
	"testing"

	forge "github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/httpreplay"
)

// These tests replay responses recorded from codeberg.org, which runs
// Forgejo. Re-record them with FORGE_RECORD=1 go test ./gitea -run TestReplay.

func TestReplayGetRepo(t *testing.T) {
	f := New("https://codeberg.org", "", httpreplay.Client(t))
	repo, err := f.Repos().Get(context.Background(), "Codeberg", "Documentation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "FullName", "Codeberg/Documentation", repo.FullName)
	assertEqual(t, "Owner", "Codeberg", repo.Owner)
	assertEqual(t, "Name", "Documentation", repo.Name)
	assertEqual(t, "HTMLURL", "https://codeberg.org/Codeberg/Documentation", repo.HTMLURL)
	assertEqual(t, "CloneURL", "https://codeberg.org/Codeberg/Documentation.git", repo.CloneURL)
	assertEqual(t, "DefaultBranch", "main", repo.DefaultBranch)
	assertEqualBool(t, "Private", false, repo.Private)
	if repo.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestReplayGetRepoNotFound(t *testing.T) {
	f := New("https://codeberg.org", "", httpreplay.Client(t))
	_, err := f.Repos().Get(context.Background(), "Codeberg", "this-repository-does-not-exist")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestReplayListBranches(t *testing.T) {
	f := New("https://codeberg.org", "", httpreplay.Client(t))
	branches, err := f.Branches().List(context.Background(), "Codeberg", "Documentation", forge.ListBranchOpts{Limit: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqualInt(t, "len(branches)", 1, len(branches))
	if branches[0].Name == "" || len(branches[0].SHA) != 40 {
		t.Errorf("branch missing name or SHA: %+v", branches[0])
	}
}

func TestReplayGetFile(t *testing.T) {
	f := New("https://codeberg.org", "", httpreplay.Client(t))
	file, err := f.Files().Get(context.Background(), "Codeberg", "Documentation", "LICENSE.md", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Name", "LICENSE.md", file.Name)
	assertEqual(t, "Path", "LICENSE.md", file.Path)
	if !strings.Contains(string(file.Content), "licensed under [CC-BY-SA 4.0]") {
		t.Errorf("Content does not look like the decoded license: %q", file.Content)
	}
}

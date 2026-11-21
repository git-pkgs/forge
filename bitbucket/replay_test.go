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

func TestReplayListCommitStatuses(t *testing.T) {
	f := New("", httpreplay.Client(t))
	statuses, err := f.CommitStatuses().List(context.Background(), "atlassian", "atlassian-connect-express", "fd61ca641c0db3ab42ad10bc1b2b7190033aee77")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(statuses) != 1 {
		t.Fatalf("got %d statuses, want 1", len(statuses))
	}
	assertEqual(t, "State", "failure", statuses[0].State)
	assertEqual(t, "Context", "default", statuses[0].Context)
	assertEqual(t, "TargetURL", "https://bitbucket.org/atlassian/atlassian-connect-express/pipelines/results/2865", statuses[0].TargetURL)
	if statuses[0].CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

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

func TestReplayListBranches(t *testing.T) {
	f := New("", httpreplay.Client(t))
	branches, err := f.Branches().List(context.Background(), "tutorials", "markdowndemo", forge.ListBranchOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(branches) != 1 {
		t.Fatalf("got %d branches, want 1", len(branches))
	}
	assertEqual(t, "Name", "master", branches[0].Name)
	assertEqual(t, "SHA", "5b6c39df3196eeeb8a36684beb9a3df854a0b5f3", branches[0].SHA)
}

func TestReplayGetRepoNotFound(t *testing.T) {
	f := New("", httpreplay.Client(t))
	_, err := f.Repos().Get(context.Background(), "tutorials", "this-repository-does-not-exist")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

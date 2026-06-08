package git

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/git-pkgs/forge"
)

type mockPRService struct {
	forges.PullRequestService
	prs []forges.PullRequest
}

func (m *mockPRService) List(ctx context.Context, owner, repo string, opts forges.ListPROpts) ([]forges.PullRequest, error) {
	return m.prs, nil
}

type mockForge struct {
	forges.Forge
	prService *mockPRService
}

func (m *mockForge) PullRequests() forges.PullRequestService {
	return m.prService
}

func TestGetOrFetchBaseBranch(t *testing.T) {
	// Create temporary directory and run git init
	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origWd)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		t.Skip("git not available, skipping test")
	}

	// We also need to configure a dummy user so git commands don't fail
	_ = exec.Command("git", "config", "user.name", "test").Run()
	_ = exec.Command("git", "config", "user.email", "test@example.com").Run()

	// 1. Test cached config
	// Set config for branch "feature-xyz"
	branch := "feature-xyz"
	wantBase := "main"
	cmdSet := exec.Command("git", "config", "branch.feature-xyz.forge-merge-base", wantBase)
	if err := cmdSet.Run(); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	// Pass nil Forge client since it should not be called when cached
	gotBase, err := GetOrFetchBaseBranch(ctx, nil, "owner", "repo", branch, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if gotBase != wantBase {
		t.Errorf("expected base branch %q, got %q", wantBase, gotBase)
	}

	// 2. Test fetching from forge
	// Delete the cached config first
	_ = exec.Command("git", "config", "--unset", "branch.feature-xyz.forge-merge-base").Run()

	mock := &mockForge{
		prService: &mockPRService{
			prs: []forges.PullRequest{
				{
					Number: 1,
					State:  "open",
					Head: forges.PRBranch{
						Ref: branch,
					},
					Base: forges.PRBranch{
						Ref: "develop",
					},
				},
			},
		},
	}

	gotBase, err = GetOrFetchBaseBranch(ctx, mock, "owner", "repo", branch, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if gotBase != "develop" {
		t.Errorf("expected base branch 'develop', got %q", gotBase)
	}

	// Verify it was cached in git config
	cachedVal, err := runGit(ctx, "config", "--get", "branch.feature-xyz.forge-merge-base")
	if err != nil {
		t.Fatalf("expected to read config, got: %v", err)
	}
	if cachedVal != "develop" {
		t.Errorf("expected cached value to be 'develop', got %q", cachedVal)
	}

	// 3. Test forceRefresh bypassing config cache
	// Reset config to 'main'
	cmdSet = exec.Command("git", "config", "branch.feature-xyz.forge-merge-base", "main")
	if err := cmdSet.Run(); err != nil {
		t.Fatal(err)
	}

	// Calling with forceRefresh=true should bypass the "main" cache and get "develop" from mock
	gotBase, err = GetOrFetchBaseBranch(ctx, mock, "owner", "repo", branch, true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if gotBase != "develop" {
		t.Errorf("expected base branch 'develop', got %q", gotBase)
	}
}

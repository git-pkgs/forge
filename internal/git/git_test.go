package git

import (
	"context"
	"os/exec"
	"testing"

	"github.com/git-pkgs/forge"
)

type mockPRService struct {
	forges.PullRequestService
	prs          []forges.PullRequest
	t            *testing.T
	expectedHead string
}

func (m *mockPRService) List(ctx context.Context, owner, repo string, opts forges.ListPROpts) ([]forges.PullRequest, error) {
	if m.expectedHead != "" && opts.Head != m.expectedHead {
		m.t.Errorf("expected opts.Head to be %q, got %q", m.expectedHead, opts.Head)
	}
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
	// Create temporary directory
	tmpDir := t.TempDir()

	// Initialize git repo in tmpDir
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skip("git not available, skipping test")
	}

	// We also need to configure a dummy user so git commands don't fail
	cmdName := exec.Command("git", "config", "user.name", "test")
	cmdName.Dir = tmpDir
	_ = cmdName.Run()

	cmdEmail := exec.Command("git", "config", "user.email", "test@example.com")
	cmdEmail.Dir = tmpDir
	_ = cmdEmail.Run()

	// 1. Test cached config
	// Set config for branch "feature-xyz"
	branch := "feature-xyz"
	wantBase := "main"
	cmdSet := exec.Command("git", "config", "branch.feature-xyz.forge-merge-base", wantBase)
	cmdSet.Dir = tmpDir
	if err := cmdSet.Run(); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	// Pass nil Forge client since it should not be called when cached
	gotBase, err := GetOrFetchBaseBranch(ctx, nil, tmpDir, "owner", "repo", branch, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if gotBase != wantBase {
		t.Errorf("expected base branch %q, got %q", wantBase, gotBase)
	}

	// 2. Test fetching from forge
	// Delete the cached config first
	cmdUnset := exec.Command("git", "config", "--unset", "branch.feature-xyz.forge-merge-base")
	cmdUnset.Dir = tmpDir
	_ = cmdUnset.Run()

	mock := &mockForge{
		prService: &mockPRService{
			t:            t,
			expectedHead: branch,
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

	gotBase, err = GetOrFetchBaseBranch(ctx, mock, tmpDir, "owner", "repo", branch, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if gotBase != "develop" {
		t.Errorf("expected base branch 'develop', got %q", gotBase)
	}

	// Verify it was cached in git config
	cachedVal, err := runGit(ctx, tmpDir, "config", "--get", "branch.feature-xyz.forge-merge-base")
	if err != nil {
		t.Fatalf("expected to read config, got: %v", err)
	}
	if cachedVal != "develop" {
		t.Errorf("expected cached value to be 'develop', got %q", cachedVal)
	}

	// 3. Test forceRefresh bypassing config cache
	// Reset config to 'main'
	cmdSet = exec.Command("git", "config", "branch.feature-xyz.forge-merge-base", "main")
	cmdSet.Dir = tmpDir
	if err := cmdSet.Run(); err != nil {
		t.Fatal(err)
	}

	// Calling with forceRefresh=true should bypass the "main" cache and get "develop" from mock
	gotBase, err = GetOrFetchBaseBranch(ctx, mock, tmpDir, "owner", "repo", branch, true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if gotBase != "develop" {
		t.Errorf("expected base branch 'develop', got %q", gotBase)
	}
}

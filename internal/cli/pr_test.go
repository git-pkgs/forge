package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/resolve"
)

func TestPRCmd(t *testing.T) {
	cmd := prCmd
	if cmd.Use != "pr" {
		t.Errorf("expected Use=pr, got %s", cmd.Use)
	}

	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "mr" {
		t.Errorf("expected alias mr, got %v", cmd.Aliases)
	}

	subcommands := cmd.Commands()
	want := map[string]bool{
		"view":    false,
		"list":    false,
		"create":  false,
		"close":   false,
		"reopen":  false,
		"edit":    false,
		"merge":   false,
		"diff":    false,
		"comment": false,
	}

	for _, sub := range subcommands {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestPRCmdAlias(t *testing.T) {
	// Verify the mr alias is registered on the root command
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "pr" {
			for _, alias := range cmd.Aliases {
				if alias == "mr" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected mr alias on pr command")
	}
}

func TestPRViewInvalidNumber(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"pr", "view", "notanumber"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-numeric PR number")
	}
	if !strings.Contains(err.Error(), "invalid PR number") {
		t.Errorf("expected 'invalid PR number' in error, got: %s", err)
	}
}

func TestPRCreateRequiresTitleAndHead(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"pr", "create"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing title")
	}
	if !strings.Contains(err.Error(), "--title is required") {
		t.Errorf("expected '--title is required' in error, got: %s", err)
	}
}

func TestPRCreateRequiresHead(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"pr", "create", "--title", "test"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing head")
	}
	if !strings.Contains(err.Error(), "--head is required") {
		t.Errorf("expected '--head is required' in error, got: %s", err)
	}
}

func TestPRMergeInvalidNumber(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"pr", "merge", "abc"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid PR number") {
		t.Errorf("expected 'invalid PR number' in error, got: %s", err)
	}
}

func TestPRDiffRequiresArg(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"pr", "diff"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing argument")
	}
}

func TestStorePRForBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	mustGit(t, dir, "init", "-q")
	mustGit(t, dir, "config", "user.email", "test@test.com")
	mustGit(t, dir, "config", "user.name", "Test")

	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", "README")
	mustGit(t, dir, "commit", "-m", "init")
	mustGit(t, dir, "checkout", "-b", "feature")

	ctx := context.Background()

	// Store PR number
	if err := storePRForBranch(ctx, "feature", 42); err != nil {
		t.Fatalf("storePRForBranch: %v", err)
	}

	// Load it back
	n, err := loadPRForBranch(ctx, "feature")
	if err != nil {
		t.Fatalf("loadPRForBranch: %v", err)
	}
	if n != 42 {
		t.Errorf("got %d, want 42", n)
	}
}

func TestLoadPRForBranchGhFormat(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	mustGit(t, dir, "init", "-q")
	mustGit(t, dir, "config", "user.email", "test@test.com")
	mustGit(t, dir, "config", "user.name", "Test")

	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", "README")
	mustGit(t, dir, "commit", "-m", "init")
	mustGit(t, dir, "checkout", "-b", "pr-branch")

	// Set up gh CLI's format: branch.<name>.merge = refs/pull/<n>/head
	mustGit(t, dir, "config", "branch.pr-branch.merge", "refs/pull/123/head")

	ctx := context.Background()
	n, err := loadPRForBranch(ctx, "pr-branch")
	if err != nil {
		t.Fatalf("loadPRForBranch: %v", err)
	}
	if n != 123 {
		t.Errorf("got %d, want 123", n)
	}
}

func TestFindPRForCurrentBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	mustGit(t, dir, "init", "-q")
	mustGit(t, dir, "config", "user.email", "test@test.com")
	mustGit(t, dir, "config", "user.name", "Test")
	mustGit(t, dir, "remote", "add", "origin", "https://github.com/testowner/testrepo.git")

	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", "README")
	mustGit(t, dir, "commit", "-m", "init")
	mustGit(t, dir, "checkout", "-b", "feature")
	mustGit(t, dir, "config", "branch.feature.remote", "origin")

	// Set up mock forge that returns a PR for our branch
	mockPR := &mockPRService{
		listResult: []forges.PullRequest{
			{
				Number: 99,
				State:  "open",
				Head: forges.PRBranch{
					Ref: "feature",
				},
			},
		},
	}
	resolve.SetTestForge(
		&mockForge{prService: mockPR},
		"testowner", "testrepo", "github.com",
	)
	t.Cleanup(resolve.ResetTestForge)

	ctx := context.Background()
	forge, owner, repo, _, err := resolve.Repo("", "")
	if err != nil {
		t.Fatalf("resolve.Repo: %v", err)
	}

	n, err := findPRForCurrentBranch(ctx, forge, owner, repo)
	if err != nil {
		t.Fatalf("findPRForCurrentBranch: %v", err)
	}
	if n != 99 {
		t.Errorf("got %d, want 99", n)
	}

	// The PR number should now be cached
	cached, err := loadPRForBranch(ctx, "feature")
	if err != nil {
		t.Fatalf("loadPRForBranch after find: %v", err)
	}
	if cached != 99 {
		t.Errorf("cached PR = %d, want 99", cached)
	}
}

func TestFindPRForCurrentBranch_OpenWinsOverClosed(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	mustGit(t, dir, "init", "-q")
	mustGit(t, dir, "config", "user.email", "test@test.com")
	mustGit(t, dir, "config", "user.name", "Test")
	mustGit(t, dir, "remote", "add", "origin", "https://github.com/testowner/testrepo.git")

	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", "README")
	mustGit(t, dir, "commit", "-m", "init")
	mustGit(t, dir, "checkout", "-b", "feature")
	mustGit(t, dir, "config", "branch.feature.remote", "origin")

	// Mock forge returns both a closed and open PR for the same branch
	mockPR := &mockPRService{
		listResult: []forges.PullRequest{
			{
				Number: 50,
				State:  "closed",
				Head: forges.PRBranch{
					Ref: "feature",
				},
			},
			{
				Number: 99,
				State:  "open",
				Head: forges.PRBranch{
					Ref: "feature",
				},
			},
		},
	}
	resolve.SetTestForge(
		&mockForge{prService: mockPR},
		"testowner", "testrepo", "github.com",
	)
	t.Cleanup(resolve.ResetTestForge)

	ctx := context.Background()
	forge, owner, repo, _, err := resolve.Repo("", "")
	if err != nil {
		t.Fatalf("resolve.Repo: %v", err)
	}

	n, err := findPRForCurrentBranch(ctx, forge, owner, repo)
	if err != nil {
		t.Fatalf("findPRForCurrentBranch: %v", err)
	}
	if n != 99 {
		t.Errorf("got %d, want 99 (the open PR should win over closed)", n)
	}

	// The open PR should be cached
	cached, err := loadPRForBranch(ctx, "feature")
	if err != nil {
		t.Fatalf("loadPRForBranch after find: %v", err)
	}
	if cached != 99 {
		t.Errorf("cached PR = %d, want 99", cached)
	}
}

func TestFindPRForCurrentBranch_ClosedPRNotCached(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	mustGit(t, dir, "init", "-q")
	mustGit(t, dir, "config", "user.email", "test@test.com")
	mustGit(t, dir, "config", "user.name", "Test")
	mustGit(t, dir, "remote", "add", "origin", "https://github.com/testowner/testrepo.git")

	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", "README")
	mustGit(t, dir, "commit", "-m", "init")
	mustGit(t, dir, "checkout", "-b", "feature")
	mustGit(t, dir, "config", "branch.feature.remote", "origin")

	// Mock forge returns only a closed PR
	mockPR := &mockPRService{
		listResult: []forges.PullRequest{
			{
				Number: 42,
				State:  "closed",
				Head: forges.PRBranch{
					Ref: "feature",
				},
			},
		},
	}
	resolve.SetTestForge(
		&mockForge{prService: mockPR},
		"testowner", "testrepo", "github.com",
	)
	t.Cleanup(resolve.ResetTestForge)

	ctx := context.Background()
	forge, owner, repo, _, err := resolve.Repo("", "")
	if err != nil {
		t.Fatalf("resolve.Repo: %v", err)
	}

	n, err := findPRForCurrentBranch(ctx, forge, owner, repo)
	if err != nil {
		t.Fatalf("findPRForCurrentBranch: %v", err)
	}
	if n != 42 {
		t.Errorf("got %d, want 42 (the closed PR should be returned)", n)
	}

	// Closed PRs should NOT be cached - loadPRForBranch should find nothing
	_, err = loadPRForBranch(ctx, "feature")
	if err == nil {
		t.Error("expected loadPRForBranch to return error for uncached closed PR, got nil")
	}
}

// mustGit runs a git command in dir (with global/system config isolated),
// failing the test on error. Passing "" for dir runs in the current directory.
func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestPRViewJSONFlagNotSupported(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "generic field",
			args: []string{"pr", "view", "--json=title", "1"},
			want: "--json is not supported; use --output json instead (field selection is not supported)\n\nTry: forge --output json pr view <number>",
		},
		{
			name: "comments field",
			args: []string{"pr", "view", "--json=comments", "1"},
			want: "--json is not supported; use --output json instead (field selection is not supported)\n\nTry: forge --output json pr view <number>\n     forge pr view --comments <number>",
		},
		{
			name: "reviews field",
			args: []string{"pr", "view", "--json=reviews", "1"},
			want: "--json is not supported; use --output json instead (field selection is not supported)\n\nTry: forge --output json pr view <number>\n     forge pr review list <number>",
		},
		{
			name: "both fields",
			args: []string{"pr", "view", "--json=reviews,comments", "1"},
			want: "--json is not supported; use --output json instead (field selection is not supported)\n\nTry: forge --output json pr view <number>\n     forge pr view --comments <number>\n     forge pr review list <number>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()
			if err == nil {
				t.Fatal("expected error for --json flag")
			}
			if err.Error() != tt.want {
				t.Errorf("unexpected error:\ngot:  %s\nwant: %s", err.Error(), tt.want)
			}
		})
	}
}

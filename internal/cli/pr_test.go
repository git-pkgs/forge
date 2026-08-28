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
	"github.com/git-pkgs/forge/internal/git"
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

func TestPRCreatePushesHeadBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	t.Chdir(dir)
	mustGit(t, "", "init", "--bare", "-q", remoteDir)
	mustGit(t, dir, "init", "-q")
	mustGit(t, dir, "config", "user.email", "test@test.com")
	mustGit(t, dir, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", "README")
	mustGit(t, dir, "commit", "-q", "-m", "initial commit")
	mustGit(t, dir, "checkout", "-q", "-b", "feature")
	mustGit(t, dir, "remote", "add", "origin", remoteDir)

	prs := &mockPRService{
		qualifiedHeads: true,
		createResult: &forges.PullRequest{
			Number:  1,
			HTMLURL: "https://example.com/pulls/1",
			Base:    forges.PRBranch{Ref: "main"},
		},
	}
	resolve.SetTestForge(&mockForge{prService: prs}, "owner", "repo", "example.com")
	t.Cleanup(resolve.ResetTestForge)
	resolve.SetRemote("origin")

	cmd := prCreateCmd()
	cmd.SetArgs([]string{"--title", "Test PR", "--head", "forkowner:feature", "--push"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pr create: %v", err)
	}

	if prs.createCalls != 1 {
		t.Fatalf("Create calls = %d, want 1", prs.createCalls)
	}
	if prs.createOpts.Head != "forkowner:feature" {
		t.Fatalf("Create head = %q, want %q", prs.createOpts.Head, "forkowner:feature")
	}
	localSHA := gitOutput(t, dir, "rev-parse", "refs/heads/feature")
	remoteSHA := gitOutput(t, "", "--git-dir", remoteDir, "rev-parse", "refs/heads/feature")
	if remoteSHA != localSHA {
		t.Fatalf("remote branch SHA = %q, want %q", remoteSHA, localSHA)
	}
	base := gitOutput(t, dir, "config", "--local", "--get", "branch.feature.forge-merge-base")
	if base != "main" {
		t.Fatalf("cached base branch = %q, want %q", base, "main")
	}
}

func TestPRCreatePushRejectsQualifiedHeadForUnsupportedForge(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	mustGit(t, dir, "init", "-q")

	prs := &mockPRService{}
	resolve.SetTestForge(&mockForge{prService: prs}, "upstream", "repo", "gitlab.com")
	t.Cleanup(resolve.ResetTestForge)

	cmd := prCreateCmd()
	cmd.SetArgs([]string{"--title", "Test PR", "--head", "forkowner:feature", "--push"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "qualified head") {
		t.Fatalf("pr create error = %v, want unsupported qualified-head error", err)
	}
	if prs.createCalls != 0 {
		t.Fatalf("Create calls = %d, want 0", prs.createCalls)
	}
}

func TestPRCreatePushRejectsForkForUnsupportedForge(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	mustGit(t, dir, "init", "-q")
	mustGit(t, dir, "remote", "add", "fork", "https://gitlab.com/forkowner/repo.git")

	prs := &mockPRService{}
	resolve.SetTestForge(&mockForge{prService: prs}, "upstream", "repo", "gitlab.com")
	t.Cleanup(resolve.ResetTestForge)
	resolve.SetRemote("fork")
	t.Cleanup(func() { resolve.SetRemote("origin") })

	cmd := prCreateCmd()
	cmd.SetArgs([]string{"--title", "Test PR", "--head", "feature", "--push"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "fork pushes are not supported") {
		t.Fatalf("pr create error = %v, want unsupported fork-push error", err)
	}
	if prs.createCalls != 0 {
		t.Fatalf("Create calls = %d, want 0", prs.createCalls)
	}
}

func TestPRCreatePushRejectsUnqualifiedForkHead(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	mustGit(t, dir, "init", "-q")
	mustGit(t, dir, "remote", "add", "fork", "https://github.com/upstream/repo.git")
	mustGit(t, dir, "remote", "set-url", "--push", "fork", "https://github.com/forkowner/repo.git")

	prs := &mockPRService{qualifiedHeads: true}
	resolve.SetTestForge(&mockForge{prService: prs}, "upstream", "repo", "github.com")
	t.Cleanup(resolve.ResetTestForge)
	resolve.SetRemote("fork")
	t.Cleanup(func() { resolve.SetRemote("origin") })

	cmd := prCreateCmd()
	cmd.SetArgs([]string{"--title", "Test PR", "--head", "feature", "--push"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "use --head forkowner:feature") {
		t.Fatalf("pr create error = %v, want qualified-head guidance", err)
	}
	if prs.createCalls != 0 {
		t.Fatalf("Create calls = %d, want 0", prs.createCalls)
	}
}

func TestValidatePushRemoteAllowsRenamedQualifiedFork(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	mustGit(t, dir, "init", "-q")
	mustGit(t, dir, "remote", "add", "fork", "https://github.com/forkowner/renamed-fork.git")
	resolve.SetRemote("fork")
	t.Cleanup(func() { resolve.SetRemote("origin") })

	err := validatePushRemote(context.Background(), "github.com", "upstream", "repo", "forkowner", "feature", true)
	if err != nil {
		t.Fatalf("validatePushRemote: %v", err)
	}
}

func TestValidatePushRemoteRejectsQualifiedSameOwnerDifferentRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	mustGit(t, dir, "init", "-q")
	mustGit(t, dir, "remote", "add", "other", "https://github.com/acme/other.git")
	resolve.SetRemote("other")
	t.Cleanup(func() { resolve.SetRemote("origin") })

	err := validatePushRemote(context.Background(), "github.com", "acme", "base", "acme", "feature", true)
	if err == nil || !strings.Contains(err.Error(), "repository \"other\" does not match target repository \"base\"") {
		t.Fatalf("validatePushRemote error = %v, want repository-mismatch error", err)
	}
}

func TestPRCreatePushRejectsMismatchedQualifiedOwner(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	mustGit(t, dir, "init", "-q")
	mustGit(t, dir, "remote", "add", "fork", "https://github.com/forkowner/repo.git")

	prs := &mockPRService{qualifiedHeads: true}
	resolve.SetTestForge(&mockForge{prService: prs}, "upstream", "repo", "github.com")
	t.Cleanup(resolve.ResetTestForge)
	resolve.SetRemote("fork")
	t.Cleanup(func() { resolve.SetRemote("origin") })

	cmd := prCreateCmd()
	cmd.SetArgs([]string{"--title", "Test PR", "--head", "otherowner:feature", "--push"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "head owner \"otherowner\" does not match") {
		t.Fatalf("pr create error = %v, want owner-mismatch error", err)
	}
	if prs.createCalls != 0 {
		t.Fatalf("Create calls = %d, want 0", prs.createCalls)
	}
}

func TestSplitPRHead(t *testing.T) {
	tests := []struct {
		head       string
		wantOwner  string
		wantBranch string
		wantErr    bool
	}{
		{head: "feature", wantBranch: "feature"},
		{head: "forkowner:feature", wantOwner: "forkowner", wantBranch: "feature"},
		{head: ":feature", wantErr: true},
		{head: "forkowner:", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.head, func(t *testing.T) {
			owner, branch, err := splitPRHead(tt.head)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("splitPRHead: %v", err)
			}
			if owner != tt.wantOwner || branch != tt.wantBranch {
				t.Fatalf("splitPRHead = %q, %q, want %q, %q", owner, branch, tt.wantOwner, tt.wantBranch)
			}
		})
	}
}

func TestPRCreatePushFailurePreventsCreate(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	mustGit(t, dir, "init", "-q")

	prs := &mockPRService{}
	resolve.SetTestForge(&mockForge{prService: prs}, "owner", "repo", "example.com")
	t.Cleanup(resolve.ResetTestForge)
	resolve.SetRemote("missing")
	t.Cleanup(func() { resolve.SetRemote("origin") })

	cmd := prCreateCmd()
	cmd.SetArgs([]string{"--title", "Test PR", "--head", "feature", "--push"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "pushing head branch") {
		t.Fatalf("pr create error = %v, want push error", err)
	}
	if prs.createCalls != 0 {
		t.Fatalf("Create calls = %d, want 0", prs.createCalls)
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
	cached, err := git.GetPRNumber(ctx, "", "feature")
	if err != nil {
		t.Fatalf("git.GetPRNumber after find: %v", err)
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
	cached, err := git.GetPRNumber(ctx, "", "feature")
	if err != nil {
		t.Fatalf("git.GetPRNumber after find: %v", err)
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

	// Closed PRs should NOT be cached - GetPRNumber should find nothing
	_, err = git.GetPRNumber(ctx, "", "feature")
	if err == nil {
		t.Error("expected git.GetPRNumber to return error for uncached closed PR, got nil")
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

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
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

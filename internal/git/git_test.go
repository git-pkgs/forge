package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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

func TestGetOrFetchBaseBranchIgnoresGlobalConfig(t *testing.T) {
	dir := initGitRepo(t)
	isolateGlobalGitConfig(t)
	mustGit(t, dir, "config", "--global", "branch.feature.forge-merge-base", "main")

	mock := &mockForge{
		prService: &mockPRService{
			t:            t,
			expectedHead: "feature",
			prs: []forges.PullRequest{
				{
					Number: 1,
					State:  "open",
					Head:   forges.PRBranch{Ref: "feature"},
					Base:   forges.PRBranch{Ref: "develop"},
				},
			},
		},
	}

	got, err := GetOrFetchBaseBranch(context.Background(), mock, dir, "owner", "repo", "feature", false)
	if err != nil {
		t.Fatalf("GetOrFetchBaseBranch: %v", err)
	}
	if got != "develop" {
		t.Fatalf("GetOrFetchBaseBranch = %q, want %q", got, "develop")
	}
}

func TestCurrentBranch(t *testing.T) {
	dir := initGitRepo(t)
	mustGit(t, dir, "checkout", "-b", "feature")

	got, err := CurrentBranch(context.Background(), dir)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if got != "feature" {
		t.Fatalf("CurrentBranch = %q, want %q", got, "feature")
	}
}

func TestPushBranch(t *testing.T) {
	dir := initGitRepo(t)
	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	mustGit(t, "", "init", "--bare", "-q", remoteDir)
	mustGit(t, dir, "config", "user.name", "Test")
	mustGit(t, dir, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", "README")
	mustGit(t, dir, "commit", "-q", "-m", "initial commit")
	mustGit(t, dir, "checkout", "-q", "-b", "feature")
	mustGit(t, dir, "remote", "add", "origin", remoteDir)

	if err := PushBranch(context.Background(), dir, "origin", "feature"); err != nil {
		t.Fatalf("PushBranch: %v", err)
	}

	localSHA, err := runGit(context.Background(), dir, "rev-parse", "refs/heads/feature")
	if err != nil {
		t.Fatalf("reading local branch: %v", err)
	}
	remoteSHA, err := runGit(context.Background(), "", "--git-dir", remoteDir, "rev-parse", "refs/heads/feature")
	if err != nil {
		t.Fatalf("reading remote branch: %v", err)
	}
	if remoteSHA != localSHA {
		t.Fatalf("remote branch SHA = %q, want %q", remoteSHA, localSHA)
	}

	upstream, err := runGit(context.Background(), dir, "rev-parse", "--abbrev-ref", "feature@{upstream}")
	if err != nil {
		t.Fatalf("reading branch upstream: %v", err)
	}
	if upstream != "origin/feature" {
		t.Fatalf("branch upstream = %q, want %q", upstream, "origin/feature")
	}
}

func TestPushBranchErrors(t *testing.T) {
	tests := []struct {
		name   string
		remote string
		branch string
		want   string
	}{
		{name: "empty remote", branch: "feature", want: "empty remote name"},
		{name: "empty branch", remote: "origin", want: "empty branch name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PushBranch(context.Background(), "", tt.remote, tt.branch)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("PushBranch error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestSetAndGetPRNumber(t *testing.T) {
	dir := initGitRepo(t)
	ctx := context.Background()

	if err := SetPRNumber(ctx, dir, "feature", 42); err != nil {
		t.Fatalf("SetPRNumber: %v", err)
	}

	got, err := GetPRNumber(ctx, dir, "feature")
	if err != nil {
		t.Fatalf("GetPRNumber: %v", err)
	}
	if got != 42 {
		t.Fatalf("GetPRNumber = %d, want 42", got)
	}
}

func TestGetPRNumberGhFormat(t *testing.T) {
	dir := initGitRepo(t)
	ctx := context.Background()

	mustGit(t, dir, "config", "branch.pr-branch.merge", "refs/pull/123/head")

	got, err := GetPRNumber(ctx, dir, "pr-branch")
	if err != nil {
		t.Fatalf("GetPRNumber: %v", err)
	}
	if got != 123 {
		t.Fatalf("GetPRNumber = %d, want 123", got)
	}
}

func TestGetPRNumberIgnoresGlobalPRNumber(t *testing.T) {
	dir := initGitRepo(t)
	isolateGlobalGitConfig(t)
	mustGit(t, dir, "config", "--global", "branch.feature.forge-pr", "99")

	if _, err := GetPRNumber(context.Background(), dir, "feature"); err == nil {
		t.Fatal("expected global forge-pr config to be ignored")
	}
}

func TestGetPRNumberIgnoresGlobalGhFormat(t *testing.T) {
	dir := initGitRepo(t)
	isolateGlobalGitConfig(t)
	mustGit(t, dir, "config", "--global", "branch.pr-branch.merge", "refs/pull/123/head")

	if _, err := GetPRNumber(context.Background(), dir, "pr-branch"); err == nil {
		t.Fatal("expected global gh-format merge config to be ignored")
	}
}

func TestGetPRNumberRejectsNonPRMergeRef(t *testing.T) {
	dir := initGitRepo(t)
	ctx := context.Background()

	mustGit(t, dir, "config", "branch.feature.merge", "refs/heads/main")

	if _, err := GetPRNumber(ctx, dir, "feature"); err == nil {
		t.Fatal("expected non-PR merge ref to be rejected")
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}
	dir := t.TempDir()
	mustGit(t, dir, "init", "-q")
	return dir
}

func isolateGlobalGitConfig(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", t.TempDir()+"/global.gitconfig")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

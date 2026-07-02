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

// mockPRService implements forges.PullRequestService for testing.
type mockPRService struct {
	pr         *forges.PullRequest
	err        error
	listResult []forges.PullRequest
	listErr    error
}

func (m *mockPRService) Get(_ context.Context, _, _ string, _ int) (*forges.PullRequest, error) {
	return m.pr, m.err
}

func (m *mockPRService) List(_ context.Context, _, _ string, _ forges.ListPROpts) ([]forges.PullRequest, error) {
	return m.listResult, m.listErr
}

func (m *mockPRService) Create(_ context.Context, _, _ string, _ forges.CreatePROpts) (*forges.PullRequest, error) {
	return nil, nil
}

func (m *mockPRService) Update(_ context.Context, _, _ string, _ int, _ forges.UpdatePROpts) (*forges.PullRequest, error) {
	return nil, nil
}

func (m *mockPRService) Close(_ context.Context, _, _ string, _ int) error {
	return nil
}

func (m *mockPRService) Reopen(_ context.Context, _, _ string, _ int) error {
	return nil
}

func (m *mockPRService) Merge(_ context.Context, _, _ string, _ int, _ forges.MergePROpts) error {
	return nil
}

func (m *mockPRService) Diff(_ context.Context, _, _ string, _ int) (string, error) {
	return "", nil
}

func (m *mockPRService) CreateComment(_ context.Context, _, _ string, _ int, _ string) (*forges.Comment, error) {
	return nil, nil
}

func (m *mockPRService) ListComments(_ context.Context, _, _ string, _ int) ([]forges.Comment, error) {
	return nil, nil
}

func (m *mockPRService) ListReactions(_ context.Context, _, _ string, _ int, _ int64) ([]forges.Reaction, error) {
	return nil, nil
}

func (m *mockPRService) AddReaction(_ context.Context, _, _ string, _ int, _ int64, _ string) (*forges.Reaction, error) {
	return nil, nil
}

func (m *mockPRService) ListURL(_ string) string {
	return ""
}

// mockForge implements forges.Forge for testing.
type mockForge struct {
	prService *mockPRService
}

func (m *mockForge) Repos() forges.RepoService                  { return nil }
func (m *mockForge) Issues() forges.IssueService                { return nil }
func (m *mockForge) PullRequests() forges.PullRequestService    { return m.prService }
func (m *mockForge) Labels() forges.LabelService                { return nil }
func (m *mockForge) Milestones() forges.MilestoneService        { return nil }
func (m *mockForge) Releases() forges.ReleaseService            { return nil }
func (m *mockForge) CI() forges.CIService                       { return nil }
func (m *mockForge) Branches() forges.BranchService             { return nil }
func (m *mockForge) DeployKeys() forges.DeployKeyService        { return nil }
func (m *mockForge) Secrets() forges.SecretService              { return nil }
func (m *mockForge) Notifications() forges.NotificationService  { return nil }
func (m *mockForge) Reviews() forges.ReviewService              { return nil }
func (m *mockForge) Files() forges.FileService                  { return nil }
func (m *mockForge) Collaborators() forges.CollaboratorService  { return nil }
func (m *mockForge) CommitStatuses() forges.CommitStatusService { return nil }
func (m *mockForge) GetRateLimit(_ context.Context) (*forges.RateLimit, error) {
	return nil, forges.ErrNotSupported
}

func (m *mockForge) ParsePath(_ []string) (*forges.ResourceRef, error) {
	return &forges.ResourceRef{
		Owner:  "testowner",
		Repo:   "testrepo",
		Type:   forges.ResourceTypePR,
		Number: 42,
	}, nil
}

// setupTestRepo creates a temporary git repository with an initial commit
// and an origin remote pointing to a fake URL.
func setupTestRepo(t *testing.T, originURL string) string {
	t.Helper()
	dir := t.TempDir()

	mustGit(t, dir, "init")
	mustGit(t, dir, "config", "user.email", "test@test.com")
	mustGit(t, dir, "config", "user.name", "Test User")

	// Create an initial commit so we have a valid HEAD
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	mustGit(t, dir, "add", "README.md")
	mustGit(t, dir, "commit", "-m", "Initial commit")
	mustGit(t, dir, "remote", "add", "origin", originURL)

	return dir
}

// setupBareRepo creates a bare git repository that can be used as a remote.
func setupBareRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	mustGit(t, dir, "init", "--bare")

	return dir
}

// pushBranchToRemote creates a branch and pushes it to a remote.
func pushBranchToRemote(t *testing.T, repoDir, remoteName, branchName string) {
	t.Helper()

	// Create a file and commit on a new branch
	testFile := filepath.Join(repoDir, branchName+".txt")
	if err := os.WriteFile(testFile, []byte("content for "+branchName+"\n"), 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	mustGit(t, repoDir, "checkout", "-b", branchName)
	mustGit(t, repoDir, "add", ".")
	mustGit(t, repoDir, "commit", "-m", "Add "+branchName)
	mustGit(t, repoDir, "push", remoteName, branchName)
	mustGit(t, repoDir, "checkout", "-")
}

// pushToRemoteRef creates a commit and pushes it to an arbitrary ref on the
// remote (e.g. refs/pull/42/head), mimicking Gitea/Forgejo PRs whose head
// branch is gone and only the pull ref remains.
func pushToRemoteRef(t *testing.T, repoDir, remoteName, ref string) {
	t.Helper()

	testFile := filepath.Join(repoDir, "pullref.txt")
	if err := os.WriteFile(testFile, []byte("content for "+ref+"\n"), 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	commands := [][]string{
		{"git", "checkout", "-b", "tmp-pushref"},
		{"git", "add", "."},
		{"git", "commit", "-m", "commit for " + ref},
		{"git", "push", remoteName, "HEAD:" + ref},
		{"git", "checkout", "-"},
		{"git", "branch", "-D", "tmp-pushref"},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git command %v failed: %v\n%s", args, err, out)
		}
	}
}

// TestPRCheckoutPullRef covers Gitea/Forgejo PRs whose head.ref is a full
// refs/pull/<n>/head ref rather than a branch name. The ref must be fetched
// as-is (not under refs/heads/) and the local branch falls back to pr-<number>.
func TestPRCheckoutPullRef(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git integration test in short mode")
	}

	checkoutCmd, _, _ := rootCmd.Find([]string{"pr", "checkout"})
	if checkoutCmd != nil {
		_ = checkoutCmd.Flags().Set("detach", "false")
		_ = checkoutCmd.Flags().Set("force", "false")
		_ = checkoutCmd.Flags().Set("branch", "")
		_ = checkoutCmd.Flags().Set("remote-name", "")
	}

	originDir := setupBareRepo(t)
	workDir := setupTestRepo(t, originDir)
	pushToRemoteRef(t, workDir, "origin", "refs/pull/42/head")
	t.Chdir(workDir)

	pr := &forges.PullRequest{
		Number: 42,
		Head:   forges.PRBranch{Ref: "refs/pull/42/head", SHA: "abc123"},
	}
	resolve.SetTestForge(
		&mockForge{prService: &mockPRService{pr: pr}},
		"testowner", "testrepo", "github.com",
	)
	t.Cleanup(resolve.ResetTestForge)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"pr", "checkout", "42"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, buf.String())
	}

	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("getting current branch: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "pr-42" {
		t.Errorf("branch: want %q, got %q", "pr-42", got)
	}
}

func TestEnsureRemote(t *testing.T) {
	tests := []struct {
		name           string
		existingURL    string
		cloneURL       string
		preferredName  string
		wantRemoteName string
		wantErr        string
	}{
		{
			name:           "exact URL match",
			existingURL:    "https://github.com/owner/repo.git",
			cloneURL:       "https://github.com/owner/repo.git",
			preferredName:  "fork",
			wantRemoteName: "origin",
		},
		{
			name:           "SSH to HTTPS same repo",
			existingURL:    "git@github.com:owner/repo.git",
			cloneURL:       "https://github.com/owner/repo.git",
			preferredName:  "fork",
			wantRemoteName: "origin",
		},
		{
			name:           "HTTPS to SSH same repo",
			existingURL:    "https://github.com/owner/repo.git",
			cloneURL:       "git@github.com:owner/repo.git",
			preferredName:  "fork",
			wantRemoteName: "origin",
		},
		{
			name:           "preferred name remote matches with different URL format",
			existingURL:    "git@github.com:contributor/repo.git",
			cloneURL:       "https://github.com/contributor/repo.git",
			preferredName:  "origin",
			wantRemoteName: "origin",
		},
		{
			name:           "different repo adds new remote",
			existingURL:    "https://github.com/owner/repo.git",
			cloneURL:       "https://github.com/other/repo.git",
			preferredName:  "other",
			wantRemoteName: "other",
		},
		{
			name:          "preferred name exists with different repo",
			existingURL:   "https://github.com/owner/repo.git",
			cloneURL:      "https://github.com/other/repo.git",
			preferredName: "origin",
			wantErr:       "already exists with a different URL",
		},
	}

	if testing.Short() {
		t.Skip("skipping git integration test in short mode")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			mustGit(t, dir, "init")
			mustGit(t, dir, "config", "user.email", "test@test.com")
			mustGit(t, dir, "config", "user.name", "Test User")
			mustGit(t, dir, "remote", "add", "origin", tt.existingURL)

			t.Chdir(dir)

			gotRemote, err := ensureRemote(context.Background(), tt.preferredName, tt.cloneURL)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if gotRemote != tt.wantRemoteName {
				t.Errorf("remote name: want %q, got %q", tt.wantRemoteName, gotRemote)
			}
		})
	}
}

func TestPRCheckout(t *testing.T) {
	tests := []struct {
		name        string
		pr          *forges.PullRequest
		args        []string
		setupOrigin bool // whether to create and push to origin
		setupFork   bool // whether to create a fork remote
		wantBranch  string
		wantRemote  string // expected remote name for fork PRs
		wantErr     string
	}{
		{
			name: "same-repo PR checks out branch",
			pr: &forges.PullRequest{
				Number: 42,
				Head: forges.PRBranch{
					Ref: "feature-branch",
					SHA: "abc123",
				},
			},
			args:        []string{"pr", "checkout", "42"},
			setupOrigin: true,
			wantBranch:  "feature-branch",
		},
		{
			name: "fork PR adds remote and checks out",
			pr: &forges.PullRequest{
				Number: 42,
				Head: forges.PRBranch{
					Ref: "feature",
					SHA: "abc123",
					Fork: &forges.ForkInfo{
						Owner:    "contributor",
						Name:     "repo",
						CloneURL: "FORK_URL_PLACEHOLDER", // will be replaced
					},
				},
			},
			args:       []string{"pr", "checkout", "42"},
			setupFork:  true,
			wantBranch: "feature",
			wantRemote: "contributor",
		},
		{
			name: "fork PR with custom remote name",
			pr: &forges.PullRequest{
				Number: 42,
				Head: forges.PRBranch{
					Ref: "feature",
					SHA: "abc123",
					Fork: &forges.ForkInfo{
						Owner:    "contributor",
						Name:     "repo",
						CloneURL: "FORK_URL_PLACEHOLDER",
					},
				},
			},
			args:       []string{"pr", "checkout", "42", "--remote-name", "upstream"},
			setupFork:  true,
			wantBranch: "feature",
			wantRemote: "upstream",
		},
		{
			name: "detach mode",
			pr: &forges.PullRequest{
				Number: 42,
				Head: forges.PRBranch{
					Ref: "feature-branch",
					SHA: "abc123",
				},
			},
			args:        []string{"pr", "checkout", "42", "--detach"},
			setupOrigin: true,
			wantBranch:  "", // detached HEAD
		},
		{
			name: "checkout with custom branch name",
			pr: &forges.PullRequest{
				Number: 42,
				Head: forges.PRBranch{
					Ref: "feature-branch",
					SHA: "abc123",
				},
			},
			args:        []string{"pr", "checkout", "42", "-b", "my-local-branch"},
			setupOrigin: true,
			wantBranch:  "my-local-branch",
		},
		{
			name: "checkout by URL",
			pr: &forges.PullRequest{
				Number: 42,
				Head: forges.PRBranch{
					Ref: "feature-branch",
					SHA: "abc123",
				},
			},
			args:        []string{"pr", "checkout", "https://github.com/testowner/testrepo/pull/42"},
			setupOrigin: true,
			wantBranch:  "feature-branch",
		},
		{
			name:    "invalid PR number",
			args:    []string{"pr", "checkout", "notanumber"},
			wantErr: "invalid PR number",
		},
		{
			name: "fork PR without clone URL",
			pr: &forges.PullRequest{
				Number: 42,
				Head: forges.PRBranch{
					Ref: "feature",
					SHA: "abc123",
					Fork: &forges.ForkInfo{
						Owner: "contributor",
						Name:  "repo",
						// CloneURL and SSHURL both empty
					},
				},
			},
			args:    []string{"pr", "checkout", "42"},
			wantErr: "no clone URL available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests that need real git operations in short mode
			if testing.Short() && (tt.setupOrigin || tt.setupFork) {
				t.Skip("skipping git integration test in short mode")
			}

			// Reset flags to defaults before each test
			// Find the checkout command and reset its flags
			checkoutCmd, _, _ := rootCmd.Find([]string{"pr", "checkout"})
			if checkoutCmd != nil {
				_ = checkoutCmd.Flags().Set("detach", "false")
				_ = checkoutCmd.Flags().Set("force", "false")
				_ = checkoutCmd.Flags().Set("branch", "")
				_ = checkoutCmd.Flags().Set("remote-name", "")
			}

			var workDir string

			// For git integration tests, set up repos
			if tt.setupOrigin || tt.setupFork {
				originDir := setupBareRepo(t)
				workDir = setupTestRepo(t, originDir)

				if tt.setupOrigin {
					branchName := tt.pr.Head.Ref
					pushBranchToRemote(t, workDir, "origin", branchName)
				}

				if tt.setupFork {
					forkDir := setupBareRepo(t)
					tt.pr.Head.Fork.CloneURL = forkDir

					branchName := tt.pr.Head.Ref
					mustGit(t, workDir, "remote", "add", "tempfork", forkDir)
					pushBranchToRemote(t, workDir, "tempfork", branchName)
					mustGit(t, workDir, "remote", "remove", "tempfork")
				}
			} else if tt.pr != nil {
				// For error tests that still need a git context, create a minimal repo
				originDir := setupBareRepo(t)
				workDir = setupTestRepo(t, originDir)
			}

			// Change to work directory for the test
			if workDir != "" {
				t.Chdir(workDir)
			}

			// Set up mock forge
			if tt.pr != nil {
				resolve.SetTestForge(
					&mockForge{prService: &mockPRService{pr: tt.pr}},
					"testowner", "testrepo", "github.com",
				)
				t.Cleanup(resolve.ResetTestForge)
			}

			// Execute command
			var buf bytes.Buffer
			rootCmd.SetOut(&buf)
			rootCmd.SetErr(&buf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			// Check error
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v\noutput: %s", err, buf.String())
			}

			if workDir == "" {
				return // no git state to verify
			}

			// Verify branch
			if tt.wantBranch != "" {
				cmd := exec.Command("git", "branch", "--show-current")
				cmd.Dir = workDir
				out, err := cmd.Output()
				if err != nil {
					t.Fatalf("getting current branch: %v", err)
				}
				gotBranch := strings.TrimSpace(string(out))
				if gotBranch != tt.wantBranch {
					t.Errorf("branch: want %q, got %q", tt.wantBranch, gotBranch)
				}
			} else {
				// Detached HEAD - verify no branch
				cmd := exec.Command("git", "branch", "--show-current")
				cmd.Dir = workDir
				out, _ := cmd.Output()
				if strings.TrimSpace(string(out)) != "" {
					t.Errorf("expected detached HEAD, but on branch %q", strings.TrimSpace(string(out)))
				}
			}

			// Verify remote for fork PRs
			if tt.wantRemote != "" {
				cmd := exec.Command("git", "remote", "-v")
				cmd.Dir = workDir
				out, err := cmd.Output()
				if err != nil {
					t.Fatalf("listing remotes: %v", err)
				}
				if !strings.Contains(string(out), tt.wantRemote) {
					t.Errorf("expected remote %q in output:\n%s", tt.wantRemote, out)
				}
			}
		})
	}
}

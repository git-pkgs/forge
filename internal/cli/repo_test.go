package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	forges "github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/config"
	"github.com/git-pkgs/forge/internal/resolve"
)

func TestRepoCmd(t *testing.T) {
	// Test that the repo command is registered and has the expected subcommands
	cmd := repoCmd
	if cmd.Use != "repo" {
		t.Errorf("expected Use=repo, got %s", cmd.Use)
	}

	subcommands := cmd.Commands()
	want := map[string]bool{
		"view":   false,
		"list":   false,
		"create": false,
		"edit":   false,
		"delete": false,
		"fork":   false,
		"clone":  false,
		"search": false,
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

func TestRootCmd(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("forge")) {
		t.Error("help output should mention forge")
	}
}

func TestRepoCreateMutuallyExclusiveVisibility(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"repo", "create", "test-repo", "--private", "--public"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for conflicting visibility flags")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' in error, got: %s", err)
	}
}

func TestRepoEditMutuallyExclusiveVisibility(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"repo", "edit", "owner/repo", "--private", "--public"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for conflicting visibility flags")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' in error, got: %s", err)
	}
}

func TestRepoListLimitCapsTotalResults(t *testing.T) {
	repos := &capturingRepoService{listRepos: testRepositories(10)}
	setupRepoCommandTest(t, repos)

	cmd := repoListCmd()
	cmd.SetArgs([]string{"octocat", "--limit", "7"})
	stdout, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("repo list: %v", err)
	}

	if !repos.listCalled {
		t.Fatal("expected Repos().List to be called")
	}
	if repos.listOwner != "octocat" {
		t.Fatalf("owner = %q, want octocat", repos.listOwner)
	}
	if repos.listOpts.Limit != 7 {
		t.Fatalf("Limit = %d, want 7", repos.listOpts.Limit)
	}
	if repos.listOpts.PerPage != 0 {
		t.Fatalf("PerPage = %d, want 0", repos.listOpts.PerPage)
	}
	assertOutputLineCount(t, stdout, 7)
}

func TestRepoSearchLimitCapsTotalResults(t *testing.T) {
	repos := &capturingRepoService{searchRepos: testRepositories(10)}
	setupRepoCommandTest(t, repos)

	cmd := repoSearchCmd()
	cmd.SetArgs([]string{"terminal", "--limit", "5", "--sort", "stars", "--order", "desc"})
	stdout, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("repo search: %v", err)
	}

	if !repos.searchCalled {
		t.Fatal("expected Repos().Search to be called")
	}
	if repos.searchOpts.Query != "terminal" {
		t.Fatalf("Query = %q, want terminal", repos.searchOpts.Query)
	}
	if repos.searchOpts.Sort != "stars" {
		t.Fatalf("Sort = %q, want stars", repos.searchOpts.Sort)
	}
	if repos.searchOpts.Order != "desc" {
		t.Fatalf("Order = %q, want desc", repos.searchOpts.Order)
	}
	if repos.searchOpts.Limit != 5 {
		t.Fatalf("Limit = %d, want 5", repos.searchOpts.Limit)
	}
	if repos.searchOpts.PerPage != 5 {
		t.Fatalf("PerPage = %d, want 5", repos.searchOpts.PerPage)
	}
	assertOutputLineCount(t, stdout, 5)
}

func TestDomainFromFlags(t *testing.T) {
	t.Chdir(t.TempDir())

	tests := []struct {
		forgeType string
		want      string
	}{
		{"", "github.com"},
		{"github", "github.com"},
		{"gitlab", "gitlab.com"},
		{"gitea", "codeberg.org"},
		{"forgejo", "codeberg.org"},
		{"bitbucket", "bitbucket.org"},
	}

	for _, tt := range tests {
		flagForgeType = tt.forgeType
		got := domainFromFlags()
		if got != tt.want {
			t.Errorf("forgeType=%q: want %q, got %q", tt.forgeType, tt.want, got)
		}
	}
	flagForgeType = "" // reset
}

func TestGitCloneArgs(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		dest     string
		gitFlags []string
		want     []string
	}{
		{
			name: "https url",
			url:  "https://github.com/owner/repo.git",
			want: []string{"clone", "--", "https://github.com/owner/repo.git"},
		},
		{
			name: "ssh url",
			url:  "git@github.com:owner/repo.git",
			want: []string{"clone", "--", "git@github.com:owner/repo.git"},
		},
		{
			name: "url that looks like a git option",
			url:  "--upload-pack=evil",
			want: []string{"clone", "--", "--upload-pack=evil"},
		},
		{
			name: "with destination",
			url:  "https://github.com/owner/repo.git",
			dest: "myrepo",
			want: []string{"clone", "--", "https://github.com/owner/repo.git", "myrepo"},
		},
		{
			name:     "with git flags",
			url:      "https://github.com/owner/repo.git",
			gitFlags: []string{"--depth", "1"},
			want:     []string{"clone", "--depth", "1", "--", "https://github.com/owner/repo.git"},
		},
		{
			name:     "with destination and git flags",
			url:      "https://github.com/owner/repo.git",
			dest:     "myrepo",
			gitFlags: []string{"--bare"},
			want:     []string{"clone", "--bare", "--", "https://github.com/owner/repo.git", "myrepo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gitCloneArgs(tt.url, tt.dest, tt.gitFlags)
			if !slices.Equal(got, tt.want) {
				t.Errorf("gitCloneArgs(%q, %q, %v) = %v, want %v", tt.url, tt.dest, tt.gitFlags, got, tt.want)
			}

			// The url must appear after the -- separator so git cannot
			// parse a server-supplied CloneURL as an option.
			sep := slices.Index(got, "--")
			urlIdx := slices.Index(got, tt.url)
			if sep == -1 {
				t.Fatal("expected -- separator in argv")
			}
			if urlIdx <= sep {
				t.Errorf("url at index %d is not after -- at index %d", urlIdx, sep)
			}
		})
	}
}

func setupRepoCommandTest(t *testing.T, repos *capturingRepoService) {
	t.Helper()
	t.Chdir(t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("FORGE_HOST", "")
	config.ResetCache()
	t.Cleanup(config.ResetCache)

	flagRepo = ""
	flagForgeType = ""
	flagHost = ""
	flagOutput = "plain"
	flagRemote = ""

	resolve.SetTestForge(&mockForge{repoService: repos}, "", "", "github.com")
	t.Cleanup(resolve.ResetTestForge)
}

type capturingRepoService struct {
	listCalled   bool
	listOwner    string
	listOpts     forges.ListRepoOpts
	listRepos    []forges.Repository
	searchCalled bool
	searchOpts   forges.SearchRepoOpts
	searchRepos  []forges.Repository
}

func (m *capturingRepoService) Get(_ context.Context, _, _ string) (*forges.Repository, error) {
	return nil, nil
}

func (m *capturingRepoService) List(_ context.Context, owner string, opts forges.ListRepoOpts) ([]forges.Repository, error) {
	m.listCalled = true
	m.listOwner = owner
	m.listOpts = opts
	return m.listRepos, nil
}

func (m *capturingRepoService) Create(_ context.Context, _ forges.CreateRepoOpts) (*forges.Repository, error) {
	return nil, nil
}

func (m *capturingRepoService) Edit(_ context.Context, _, _ string, _ forges.EditRepoOpts) (*forges.Repository, error) {
	return nil, nil
}

func (m *capturingRepoService) Delete(_ context.Context, _, _ string) error {
	return nil
}

func (m *capturingRepoService) Fork(_ context.Context, _, _ string, _ forges.ForkRepoOpts) (*forges.Repository, error) {
	return nil, nil
}

func (m *capturingRepoService) ListForks(_ context.Context, _, _ string, _ forges.ListForksOpts) ([]forges.Repository, error) {
	return nil, nil
}

func (m *capturingRepoService) ListTags(_ context.Context, _, _ string) ([]forges.Tag, error) {
	return nil, nil
}

func (m *capturingRepoService) ListContributors(_ context.Context, _, _ string) ([]forges.Contributor, error) {
	return nil, nil
}

func (m *capturingRepoService) Search(_ context.Context, opts forges.SearchRepoOpts) ([]forges.Repository, error) {
	m.searchCalled = true
	m.searchOpts = opts
	return m.searchRepos, nil
}

func testRepositories(count int) []forges.Repository {
	repos := make([]forges.Repository, count)
	for i := range repos {
		repos[i] = forges.Repository{FullName: fmt.Sprintf("octocat/repo-%02d", i+1)}
	}
	return repos
}

func captureStdout(t *testing.T, run func() error) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stdout pipe: %v", err)
	}
	os.Stdout = w

	runErr := run()
	closeErr := w.Close()
	os.Stdout = oldStdout

	out, readErr := io.ReadAll(r)
	if err := r.Close(); err != nil {
		t.Fatalf("closing stdout reader: %v", err)
	}
	if closeErr != nil {
		t.Fatalf("closing stdout writer: %v", closeErr)
	}
	if readErr != nil {
		t.Fatalf("reading stdout: %v", readErr)
	}
	return string(out), runErr
}

func assertOutputLineCount(t *testing.T, stdout string, want int) {
	t.Helper()

	got := 0
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		if line != "" {
			got++
		}
	}
	if got != want {
		t.Fatalf("printed %d lines, want %d; stdout:\n%s", got, want, stdout)
	}
}

func (m *capturingRepoService) SettingsURL(repoHTMLURL string) string {
	return repoHTMLURL + "/settings"
}

func (m *capturingRepoService) WikiURL(repoHTMLURL string) string {
	return repoHTMLURL + "/wiki"
}

func (m *capturingRepoService) ActionsURL(repoHTMLURL string) string {
	return repoHTMLURL + "/actions"
}

func (m *capturingRepoService) ReleasesURL(repoHTMLURL string) string {
	return repoHTMLURL + "/releases"
}

func (m *capturingRepoService) BlobURL(repoHTMLURL, ref, path string) string {
	return repoHTMLURL + "/blob/" + ref + "/" + path
}

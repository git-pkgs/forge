package resolve

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-pkgs/forge/internal/config"
)

func TestMapSSHHost(t *testing.T) {
	config.ResetCache()
	defer config.ResetCache()

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgDir := filepath.Join(dir, "forge")
	_ = os.MkdirAll(cfgDir, 0700)
	_ = os.WriteFile(filepath.Join(cfgDir, "config"), []byte(`[gitlab.test]
type = gitlab
ssh_host = ssh.gitlab.test
`), 0600)

	tests := []struct {
		in   string
		want string
	}{
		// remote URL host matches a configured ssh_host: map to the API host
		{"ssh.gitlab.test", "gitlab.test"},
		// no mapping: pass through unchanged
		{"github.com", "github.com"},
		{"gitlab.test", "gitlab.test"},
	}

	for _, tt := range tests {
		got := mapSSHHost(tt.in)
		if got != tt.want {
			t.Errorf("mapSSHHost(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMapSSHHostNoConfig(t *testing.T) {
	config.ResetCache()
	defer config.ResetCache()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// With no config file, the domain passes through unchanged.
	got := mapSSHHost("ssh.gitlab.test")
	if got != "ssh.gitlab.test" {
		t.Errorf("with no config, expected passthrough, got %q", got)
	}
}

func TestTokenForDomain(t *testing.T) {
	// With no env vars set, should return empty
	got := TokenForDomain("example.com")
	if got != "" {
		t.Errorf("expected empty token, got %q", got)
	}

	// FORGE_TOKEN is a fallback for any domain
	t.Setenv("FORGE_TOKEN", "forge-tok")
	got = TokenForDomain("github.com")
	if got != "forge-tok" {
		t.Errorf("expected forge-tok, got %q", got)
	}
	t.Setenv("FORGE_TOKEN", "")

	// GitHub-specific tokens
	t.Setenv("GITHUB_TOKEN", "gh-tok")
	got = TokenForDomain("github.com")
	if got != "gh-tok" {
		t.Errorf("expected gh-tok, got %q", got)
	}
	t.Setenv("GITHUB_TOKEN", "")

	t.Setenv("GH_TOKEN", "gh2-tok")
	got = TokenForDomain("github.com")
	if got != "gh2-tok" {
		t.Errorf("expected gh2-tok, got %q", got)
	}
	t.Setenv("GH_TOKEN", "")

	// GitLab
	t.Setenv("GITLAB_TOKEN", "gl-tok")
	got = TokenForDomain("gitlab.com")
	if got != "gl-tok" {
		t.Errorf("expected gl-tok, got %q", got)
	}
	t.Setenv("GITLAB_TOKEN", "")
}

func TestTokenForDomainEnvSpecificOverridesFallback(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "gh-specific")
	t.Setenv("FORGE_TOKEN", "forge-fallback")

	got := TokenForDomainEnv("github.com")
	if got != "gh-specific" {
		t.Errorf("expected domain-specific GITHUB_TOKEN to win, got %q", got)
	}
}

func TestTokenForDomainEnvFallbackToForgeToken(t *testing.T) {
	t.Setenv("FORGE_TOKEN", "forge-fallback")

	got := TokenForDomainEnv("github.com")
	if got != "forge-fallback" {
		t.Errorf("expected FORGE_TOKEN fallback, got %q", got)
	}
}

func TestTokenForDomainEnvFallbackForUnknownDomain(t *testing.T) {
	t.Setenv("FORGE_TOKEN", "forge-fallback")

	got := TokenForDomainEnv("custom.example.com")
	if got != "forge-fallback" {
		t.Errorf("expected FORGE_TOKEN for unknown domain, got %q", got)
	}
}

func TestDomain(t *testing.T) {
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
		{"unknown", "github.com"},
	}

	for _, tt := range tests {
		got := Domain(tt.forgeType)
		if got != tt.want {
			t.Errorf("Domain(%q) = %q, want %q", tt.forgeType, got, tt.want)
		}
	}
}

func TestDomainWithForgeHost(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("FORGE_HOST", "git.example.com")

	got := Domain("github")
	if got != "git.example.com" {
		t.Errorf("expected FORGE_HOST override, got %q", got)
	}

	got = Domain("")
	if got != "git.example.com" {
		t.Errorf("expected FORGE_HOST override for empty type, got %q", got)
	}
}

func TestDomainFallsBackToGitRemote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	t.Chdir(t.TempDir())
	mustGit(t, "init", "-q")
	mustGit(t, "remote", "add", "origin", "https://gitea.com/someone/project.git")

	got := Domain("")
	if got != "gitea.com" {
		t.Errorf("expected domain from git remote, got %q", got)
	}
}

func TestDomainExplicitForgeTypeOverridesRemote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	t.Chdir(t.TempDir())
	mustGit(t, "init", "-q")
	mustGit(t, "remote", "add", "origin", "https://gitea.com/someone/project.git")

	got := Domain("gitlab")
	if got != "gitlab.com" {
		t.Errorf("expected --forge-type to override remote, got %q", got)
	}
}

func TestDomainHostOverride(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	t.Chdir(t.TempDir())
	mustGit(t, "init", "-q")
	mustGit(t, "remote", "add", "origin", "https://github.com/someone/project.git")
	t.Setenv("FORGE_HOST", "env.example.com")

	old := hostOverride
	defer func() { hostOverride = old }()
	SetHost("flag.example.com")

	got := Domain("gitlab")
	if got != "flag.example.com" {
		t.Errorf("expected --host to override everything, got %q", got)
	}
}

func TestForgeTypeOverrideSkipsDetection(t *testing.T) {
	config.ResetCache()
	defer config.ResetCache()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	old := forgeTypeOverride
	defer func() { forgeTypeOverride = old }()
	SetForgeType("gitea")

	f, err := ForgeForDomain("forge.invalid")
	if err != nil {
		t.Fatalf("--forge-type should skip network detection, got: %v", err)
	}
	if f == nil {
		t.Fatal("expected a forge instance")
	}
}

func TestSetForgeType(t *testing.T) {
	old := forgeTypeOverride
	defer func() { forgeTypeOverride = old }()

	SetForgeType("gitea")
	if forgeTypeOverride != "gitea" {
		t.Errorf("SetForgeType did not update forgeTypeOverride, got %q", forgeTypeOverride)
	}

	SetForgeType("")
	if forgeTypeOverride != "gitea" {
		t.Errorf("SetForgeType(\"\") should be a no-op, got %q", forgeTypeOverride)
	}
}

func TestSetHost(t *testing.T) {
	old := hostOverride
	defer func() { hostOverride = old }()

	SetHost("gitea.com")
	if hostOverride != "gitea.com" {
		t.Errorf("SetHost did not update hostOverride, got %q", hostOverride)
	}

	SetHost("")
	if hostOverride != "gitea.com" {
		t.Errorf("SetHost(\"\") should be a no-op, got %q", hostOverride)
	}
}

func TestRemoteDefaultsToOrigin(t *testing.T) {
	if remoteName != "origin" {
		t.Errorf("default remote should be origin, got %q", remoteName)
	}
}

func TestSetRemote(t *testing.T) {
	old := remoteName
	defer func() { remoteName = old }()

	SetRemote("upstream")
	if remoteName != "upstream" {
		t.Errorf("SetRemote did not update remoteName, got %q", remoteName)
	}

	// Empty string should leave the default alone so callers can pass
	// a flag value unconditionally without resetting to "".
	SetRemote("")
	if remoteName != "upstream" {
		t.Errorf("SetRemote(\"\") should be a no-op, got %q", remoteName)
	}
}

func TestRemoteSelectsCorrectGitURL(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	mustGit(t, "init", "-q")
	mustGit(t, "remote", "add", "origin", "https://gitea.example.com/owner/origin-repo.git")
	mustGit(t, "remote", "add", "mirror", "https://github.com/owner/mirror-repo.git")

	old := remoteName
	defer func() { remoteName = old }()

	tests := []struct {
		remote     string
		wantDomain string
		wantRepo   string
	}{
		{"origin", "gitea.example.com", "origin-repo"},
		{"mirror", "github.com", "mirror-repo"},
	}

	for _, tt := range tests {
		t.Run(tt.remote, func(t *testing.T) {
			SetRemote(tt.remote)
			domain, owner, repo, err := resolveRemote()
			if err != nil {
				t.Fatalf("resolveRemote: %v", err)
			}
			if domain != tt.wantDomain {
				t.Errorf("domain = %q, want %q", domain, tt.wantDomain)
			}
			if owner != "owner" {
				t.Errorf("owner = %q, want owner", owner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

func TestRemoteUnknownNameError(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	mustGit(t, "init", "-q")
	mustGit(t, "remote", "add", "origin", "https://github.com/owner/repo.git")

	old := remoteName
	defer func() { remoteName = old }()

	SetRemote("doesnotexist")
	_, _, _, err := resolveRemote()
	if err == nil {
		t.Fatal("expected error for unknown remote")
	}
	if !strings.Contains(err.Error(), "doesnotexist") {
		t.Errorf("error should mention the remote name, got: %v", err)
	}
}

func mustGit(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestRepoFromFlag(t *testing.T) {
	config.ResetCache()
	defer config.ResetCache()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Chdir(t.TempDir())

	tests := []struct {
		name       string
		flagRepo   string
		wantDomain string
		wantOwner  string
		wantRepo   string
		wantErr    bool
	}{
		{
			name:       "owner/repo uses default domain",
			flagRepo:   "owner/repo",
			wantDomain: "github.com",
			wantOwner:  "owner",
			wantRepo:   "repo",
		},
		{
			name:       "host/owner/repo",
			flagRepo:   "codeberg.org/owner/repo",
			wantDomain: "codeberg.org",
			wantOwner:  "owner",
			wantRepo:   "repo",
		},
		{
			name:     "single part is invalid",
			flagRepo: "repo",
			wantErr:  true,
		},
		{
			name:     "empty is invalid",
			flagRepo: "",
			wantErr:  true,
		},
		{
			name:     "too many parts is invalid",
			flagRepo: "host/group/subgroup/repo",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, owner, repo, domain, err := repoFromFlag(tt.flagRepo, "")
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if f == nil {
				t.Error("expected forge instance, got nil")
			}
			if domain != tt.wantDomain {
				t.Errorf("domain = %q, want %q", domain, tt.wantDomain)
			}
			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

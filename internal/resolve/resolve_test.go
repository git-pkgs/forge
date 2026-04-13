package resolve

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

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

func TestDomainFromForgeType(t *testing.T) {
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
		got := DomainFromForgeType(tt.forgeType)
		if got != tt.want {
			t.Errorf("DomainFromForgeType(%q) = %q, want %q", tt.forgeType, got, tt.want)
		}
	}
}

func TestDomainFromForgeTypeWithForgeHost(t *testing.T) {
	t.Setenv("FORGE_HOST", "git.example.com")

	got := DomainFromForgeType("github")
	if got != "git.example.com" {
		t.Errorf("expected FORGE_HOST override, got %q", got)
	}

	got = DomainFromForgeType("")
	if got != "git.example.com" {
		t.Errorf("expected FORGE_HOST override for empty type, got %q", got)
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

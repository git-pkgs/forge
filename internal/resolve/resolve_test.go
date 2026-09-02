package resolve

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/config"
)

func TestResourceFromURLUsesSchemeAndPort(t *testing.T) {
	config.ResetCache()
	defer config.ResetCache()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Forgejo-Version", "7.0.0")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f, domain, ref, err := ResourceFromURL(srv.URL + "/owner/repo/pulls/42")
	if err != nil {
		t.Fatalf("ResourceFromURL: %v", err)
	}
	if want := strings.TrimPrefix(srv.URL, "http://"); domain != want {
		t.Errorf("domain = %q, want %q", domain, want)
	}
	if ref.Owner != "owner" || ref.Repo != "repo" || ref.Type != forges.ResourceTypePR || ref.Number != 42 {
		t.Errorf("unexpected resource ref: %+v", ref)
	}
	provider, ok := f.(forges.APIBaseURLProvider)
	if !ok {
		t.Fatal("forge does not implement APIBaseURLProvider")
	}
	if got, want := provider.APIBaseURL(), srv.URL+"/api/v1"; got != want {
		t.Errorf("APIBaseURL = %q, want %q", got, want)
	}
}

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

func clearTokenEnv(t *testing.T) {
	t.Helper()
	for _, v := range []string{
		"GITHUB_TOKEN", "GH_TOKEN",
		"GITLAB_TOKEN", "GLAB_TOKEN",
		"FORGEJO_TOKEN", "GITEA_TOKEN", "BITBUCKET_TOKEN", "TANGLED_TOKEN",
		"FORGE_TOKEN",
	} {
		t.Setenv(v, "")
	}
}

func TestTokenForDomain(t *testing.T) {
	clearTokenEnv(t)

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

	// Codeberg (Forgejo / Gitea)
	t.Setenv("GITEA_TOKEN", "gitea-tok")
	got = TokenForDomain("codeberg.org")
	if got != "gitea-tok" {
		t.Errorf("expected gitea-tok, got %q", got)
	}
	t.Setenv("GITEA_TOKEN", "")

	t.Setenv("FORGEJO_TOKEN", "forgejo-tok")
	got = TokenForDomain("codeberg.org")
	if got != "forgejo-tok" {
		t.Errorf("expected forgejo-tok, got %q", got)
	}

	// FORGEJO_TOKEN should override GITEA_TOKEN
	t.Setenv("GITEA_TOKEN", "gitea-tok")
	got = TokenForDomain("codeberg.org")
	if got != "forgejo-tok" {
		t.Errorf("expected forgejo-tok to override gitea-tok, got %q", got)
	}
	t.Setenv("FORGEJO_TOKEN", "")
	t.Setenv("GITEA_TOKEN", "")

	// Tangled
	t.Setenv("TANGLED_TOKEN", "tangled-tok")
	got = TokenForDomain("tangled.org")
	if got != "tangled-tok" {
		t.Errorf("expected tangled-tok, got %q", got)
	}
	t.Setenv("TANGLED_TOKEN", "")
}

func TestTokenForDomainLogsFailingCommand(t *testing.T) {
	clearTokenEnv(t)

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	config.ResetCache()
	defer config.ResetCache()

	cfgDir := filepath.Join(dir, "forge")
	_ = os.MkdirAll(cfgDir, 0700)
	_ = os.WriteFile(filepath.Join(cfgDir, "config"), []byte(`[example.com]
token-cmd = false
`), 0600)

	// Redirect os.Stderr to capture the log line.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStderr := os.Stderr
	os.Stderr = w

	got := TokenForDomain("example.com")

	_ = w.Close()
	os.Stderr = origStderr

	out, _ := io.ReadAll(r)
	_ = r.Close()

	if got != "" {
		t.Errorf("expected empty token on command failure, got %q", got)
	}
	logged := string(out)
	if !strings.Contains(logged, "example.com") {
		t.Errorf("expected domain in error log, got: %s", logged)
	}
	if !strings.Contains(logged, "failed") {
		t.Errorf("expected \"failed\" in error log, got: %s", logged)
	}
}

func TestTokenForDomainEnvSpecificOverridesFallback(t *testing.T) {
	clearTokenEnv(t)
	t.Setenv("GITHUB_TOKEN", "gh-specific")
	t.Setenv("FORGE_TOKEN", "forge-fallback")

	got := TokenForDomainEnv("github.com")
	if got != "gh-specific" {
		t.Errorf("expected domain-specific GITHUB_TOKEN to win, got %q", got)
	}
}

func TestTokenForDomainEnvFallbackToForgeToken(t *testing.T) {
	clearTokenEnv(t)
	t.Setenv("FORGE_TOKEN", "forge-fallback")

	got := TokenForDomainEnv("github.com")
	if got != "forge-fallback" {
		t.Errorf("expected FORGE_TOKEN fallback, got %q", got)
	}
}

func TestTokenForDomainEnvFallbackForUnknownDomain(t *testing.T) {
	clearTokenEnv(t)
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
		{"gerrit", ""},
		{"tangled", "tangled.org"},
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

func TestForgeForDomainRequiresDomain(t *testing.T) {
	_, err := ForgeForDomain("")
	if err == nil {
		t.Fatal("expected error for empty domain")
	}
	if !strings.Contains(err.Error(), "domain is required") {
		t.Fatalf("error = %v, want domain-focused message", err)
	}
	if strings.Contains(err.Error(), "forge type") {
		t.Fatalf("error should not mention forge type: %v", err)
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

func TestSetHostWithScheme(t *testing.T) {
	oldH, oldS := hostOverride, schemeOverride
	defer func() { hostOverride, schemeOverride = oldH, oldS }()

	SetHost("HTTP://172.30.0.10:3000/forge/")
	if hostOverride != "172.30.0.10:3000" {
		t.Errorf("SetHost with URL should store bare host, got %q", hostOverride)
	}
	if schemeOverride != "http" {
		t.Errorf("SetHost with URL should record scheme, got %q", schemeOverride)
	}

	// Bare host clears any prior scheme override so a later --host without a
	// scheme does not inherit the previous one.
	SetHost("gitea.example.com")
	if schemeOverride != "" {
		t.Errorf("SetHost with bare host should clear scheme, got %q", schemeOverride)
	}
}

func TestBaseURLForDefaultsToHTTPS(t *testing.T) {
	config.ResetCache()
	defer config.ResetCache()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("FORGE_HOST", "")

	oldH, oldS := hostOverride, schemeOverride
	defer func() { hostOverride, schemeOverride = oldH, oldS }()
	hostOverride, schemeOverride = "", ""

	if got := baseURLFor("gitea.example.com"); got != "https://gitea.example.com" {
		t.Errorf("baseURLFor default = %q, want https://gitea.example.com", got)
	}
}

func TestBaseURLForFromHostFlag(t *testing.T) {
	config.ResetCache()
	defer config.ResetCache()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("FORGE_HOST", "")

	oldH, oldS := hostOverride, schemeOverride
	defer func() { hostOverride, schemeOverride = oldH, oldS }()
	SetHost("http://172.30.0.10:3000")

	if got := baseURLFor("172.30.0.10:3000"); got != "http://172.30.0.10:3000" {
		t.Errorf("baseURLFor from --host = %q, want http://172.30.0.10:3000", got)
	}
	// Scheme override only applies to the host it was given for.
	if got := baseURLFor("codeberg.org"); got != "https://codeberg.org" {
		t.Errorf("baseURLFor other domain = %q, want https://codeberg.org", got)
	}
}

func TestBaseURLForFromEnv(t *testing.T) {
	config.ResetCache()
	defer config.ResetCache()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("FORGE_HOST", "http://forgejo.local:3000")

	oldH, oldS := hostOverride, schemeOverride
	defer func() { hostOverride, schemeOverride = oldH, oldS }()
	hostOverride, schemeOverride = "", ""

	if got := Domain(""); got != "forgejo.local:3000" {
		t.Errorf("Domain with URL FORGE_HOST = %q, want forgejo.local:3000", got)
	}
	if got := baseURLFor("forgejo.local:3000"); got != "http://forgejo.local:3000" {
		t.Errorf("baseURLFor from FORGE_HOST = %q, want http://forgejo.local:3000", got)
	}
	if got := baseURLFor("codeberg.org"); got != "https://codeberg.org" {
		t.Errorf("baseURLFor other domain = %q, want https://codeberg.org", got)
	}
}

func TestBaseURLForFromConfig(t *testing.T) {
	config.ResetCache()
	defer config.ResetCache()

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("FORGE_HOST", "")
	cfgDir := filepath.Join(dir, "forge")
	_ = os.MkdirAll(cfgDir, 0700)
	_ = os.WriteFile(filepath.Join(cfgDir, "config"), []byte(`[172.30.0.10:3000]
type = forgejo
scheme = http
`), 0600)

	oldH, oldS := hostOverride, schemeOverride
	defer func() { hostOverride, schemeOverride = oldH, oldS }()
	hostOverride, schemeOverride = "", ""

	if got := baseURLFor("172.30.0.10:3000"); got != "http://172.30.0.10:3000" {
		t.Errorf("baseURLFor from config = %q, want http://172.30.0.10:3000", got)
	}
}

func TestForgeForDomainHTTPScheme(t *testing.T) {
	config.ResetCache()
	defer config.ResetCache()

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("FORGE_HOST", "")
	cfgDir := filepath.Join(dir, "forge")
	_ = os.MkdirAll(cfgDir, 0700)
	_ = os.WriteFile(filepath.Join(cfgDir, "config"), []byte(`[forgejo.local:3000]
type = forgejo
scheme = http
`), 0600)

	oldH, oldS := hostOverride, schemeOverride
	defer func() { hostOverride, schemeOverride = oldH, oldS }()
	hostOverride, schemeOverride = "", ""

	f, err := ForgeForDomain("forgejo.local:3000")
	if err != nil {
		t.Fatalf("ForgeForDomain: %v", err)
	}
	p, ok := f.(forges.APIBaseURLProvider)
	if !ok {
		t.Fatalf("forge does not implement APIBaseURLProvider")
	}
	if got := p.APIBaseURL(); got != "http://forgejo.local:3000/api/v1" {
		t.Errorf("APIBaseURL = %q, want http://forgejo.local:3000/api/v1", got)
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
	mustGit(t, "remote", "add", "local", "http://172.30.0.10:3000/owner/local-repo.git")

	old := remoteName
	defer func() { remoteName = old }()

	tests := []struct {
		remote     string
		wantDomain string
		wantRepo   string
	}{
		{"origin", "gitea.example.com", "origin-repo"},
		{"mirror", "github.com", "mirror-repo"},
		{"local", "172.30.0.10:3000", "local-repo"},
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

func TestRepoResolvesGitRemoteURLs(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	config.ResetCache()
	t.Cleanup(config.ResetCache)
	if err := os.WriteFile(filepath.Join(dir, ".forge"), []byte("[2001:db8::1]\ntype = github\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	mustGit(t, "init", "-q")
	mustGit(t, "remote", "add", "origin", "git@github.com:owner/repo.git")

	old := remoteName
	t.Cleanup(func() { remoteName = old })
	SetRemote("origin")

	tests := []struct {
		name      string
		remoteURL string
		domain    string
		owner     string
		wantErr   bool
	}{
		{name: "arbitrary SSH user", remoteURL: "org-12345@github.com:owner/repo.git", domain: "github.com", owner: "owner"},
		{name: "numeric owner", remoteURL: "git@github.com:123/repo.git", domain: "github.com", owner: "123"},
		{name: "HTTPS userinfo", remoteURL: "https://user:token@github.com/owner/repo.git", domain: "github.com", owner: "owner"},
		{name: "bracketed IPv6", remoteURL: "ssh://git@[2001:db8::1]:2222/owner/repo.git", domain: "2001:db8::1", owner: "owner"},
		{name: "bracketed IPv6 SCP", remoteURL: "git@[2001:db8::1]:owner/repo.git", domain: "2001:db8::1", owner: "owner"},
		{name: "empty host", remoteURL: ":owner/repo", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mustGit(t, "remote", "set-url", "origin", tt.remoteURL)

			_, owner, repo, domain, err := Repo("", "")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Repo: %v", err)
			}
			if domain != tt.domain {
				t.Errorf("domain = %q, want %q", domain, tt.domain)
			}
			if owner != tt.owner {
				t.Errorf("owner = %q, want %q", owner, tt.owner)
			}
			if repo != "repo" {
				t.Errorf("repo = %q, want repo", repo)
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

func TestOwnerForBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	mustGit(t, "init", "-q")
	mustGit(t, "config", "user.email", "test@test.com")
	mustGit(t, "config", "user.name", "Test")
	mustGit(t, "remote", "add", "origin", "https://github.com/mainowner/repo.git")
	mustGit(t, "remote", "add", "fork", "https://github.com/forkowner/repo.git")

	// Create initial commit so we can create branches
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, "add", "README")
	mustGit(t, "commit", "-m", "init")

	// Create a branch tracking origin
	mustGit(t, "checkout", "-b", "origin-branch")
	mustGit(t, "config", "branch.origin-branch.remote", "origin")

	// Create a branch tracking fork
	mustGit(t, "checkout", "-b", "fork-branch")
	mustGit(t, "config", "branch.fork-branch.remote", "fork")

	tests := []struct {
		branch    string
		wantOwner string
		wantErr   bool
	}{
		{"origin-branch", "mainowner", false},
		{"fork-branch", "forkowner", false},
		{"nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			owner, err := OwnerForBranch(context.Background(), tt.branch)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
		})
	}
}

func TestPushRemoteRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	mustGit(t, "init", "-q")
	mustGit(t, "remote", "add", "origin", "https://github.com/mainowner/project.git")
	mustGit(t, "remote", "set-url", "--push", "origin", "git@github.com:forkowner/project.git")

	domain, owner, repo, err := PushRemoteRepo(context.Background(), "origin")
	if err != nil {
		t.Fatalf("PushRemoteRepo: %v", err)
	}
	if domain != "github.com" || owner != "forkowner" || repo != "project" {
		t.Fatalf("PushRemoteRepo = %q, %q, %q, want %q, %q, %q", domain, owner, repo, "github.com", "forkowner", "project")
	}
}

func TestPushRemoteRepoRejectsLocalPath(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	dir := t.TempDir()
	t.Chdir(dir)
	mustGit(t, "init", "-q")
	mustGit(t, "remote", "add", "origin", filepath.Join(t.TempDir(), "remote.git"))

	if _, _, _, err := PushRemoteRepo(context.Background(), "origin"); err == nil {
		t.Fatal("expected local-path remote to be rejected")
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

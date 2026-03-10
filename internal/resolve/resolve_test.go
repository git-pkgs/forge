package resolve

import (
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

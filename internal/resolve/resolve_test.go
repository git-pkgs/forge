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

	// FORGE_TOKEN takes priority
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

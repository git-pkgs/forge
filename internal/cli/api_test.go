package cli

import (
	"os"
	"path/filepath"
	"testing"

	forges "github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/bitbucket"
	"github.com/git-pkgs/forge/gitea"
	ghforge "github.com/git-pkgs/forge/github"
	glforge "github.com/git-pkgs/forge/gitlab"
	"github.com/git-pkgs/forge/internal/config"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/git-pkgs/forge/tangled"
)

func TestAPIBaseURLFromForge(t *testing.T) {
	tests := []struct {
		name string
		f    forges.Forge
		want string
	}{
		{
			name: "github",
			f:    ghforge.New("", nil),
			want: "https://api.github.com",
		},
		{
			name: "github enterprise",
			f:    ghforge.NewWithBase("https://github.enterprise.example.org", "", nil),
			want: "https://github.enterprise.example.org/api/v3",
		},
		{
			name: "gitlab",
			f:    glforge.New("https://mylab.example.org", "", nil),
			want: "https://mylab.example.org/api/v4",
		},
		{
			name: "gitea",
			f:    gitea.New("https://gitea.example.org", "", nil),
			want: "https://gitea.example.org/api/v1",
		},
		{
			name: "forgejo",
			f:    gitea.New("https://forgejo.example.org", "", nil),
			want: "https://forgejo.example.org/api/v1",
		},
		{
			name: "bitbucket",
			f:    bitbucket.New("", nil),
			want: "https://api.bitbucket.org/2.0",
		},
		{
			name: "tangled",
			f:    tangled.New("https://tangled.example.org", "", nil),
			want: "https://tangled.example.org/xrpc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apiBaseURL(tt.f, "fallback.example.org")
			if got != tt.want {
				t.Fatalf("APIBaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIBaseURLFallsBackToLegacyDomainHeuristics(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   string
	}{
		{
			name:   "github",
			domain: "github.example.org",
			want:   "https://api.github.example.org",
		},
		{
			name:   "gitlab",
			domain: "gitlab.example.org",
			want:   "https://gitlab.example.org/api/v4",
		},
		{
			name:   "bitbucket",
			domain: "bitbucket.example.org",
			want:   "https://api.bitbucket.org/2.0",
		},
		{
			name:   "default",
			domain: "forge.example.org",
			want:   "https://forge.example.org/api/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apiBaseURL(&mockForge{}, tt.domain)
			if got != tt.want {
				t.Fatalf("apiBaseURL(..., %q) = %q, want %q", tt.domain, got, tt.want)
			}
		})
	}
}

func TestAPIBaseURLUsesLegacyFallbackForUnknownDomainWithoutConfiguredForgeType(t *testing.T) {
	got := apiBaseURL(&mockForge{}, "mylab.example.org")
	want := "https://mylab.example.org/api/v1"
	if got != want {
		t.Fatalf("apiBaseURL(..., %q) = %q, want %q", "mylab.example.org", got, want)
	}
}

func TestAPIBaseURLUsesConfiguredForgeTypeForUnknownDomain(t *testing.T) {
	config.ResetCache()
	defer config.ResetCache()

	xdgConfigHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)

	configDir := filepath.Join(xdgConfigHome, "forge")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config")
	if err := os.WriteFile(configPath, []byte(`[mylab.example.org]
type = gitlab
`), 0600); err != nil {
		t.Fatal(err)
	}

	forge, _, _, domain, err := resolve.Repo("mylab.example.org/imt/deployments", "")
	if err != nil {
		t.Fatal(err)
	}

	got := apiBaseURL(forge, domain)
	want := "https://mylab.example.org/api/v4"
	if got != want {
		t.Fatalf("apiBaseURL(..., %q) = %q, want %q", domain, got, want)
	}
}

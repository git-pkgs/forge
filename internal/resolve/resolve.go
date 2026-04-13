package resolve

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/bitbucket"
	"github.com/git-pkgs/forge/gitea"
	ghforge "github.com/git-pkgs/forge/github"
	glforge "github.com/git-pkgs/forge/gitlab"
	"github.com/git-pkgs/forge/internal/config"
)

const ownerRepoParts = 2

var remoteName = "origin"

// SetRemote sets which git remote to read when resolving the current
// repository. The CLI calls this from the --remote persistent flag.
// An empty string is ignored so callers can pass the flag value
// unconditionally.
func SetRemote(name string) {
	if name != "" {
		remoteName = name
	}
}

var builders = forges.ForgeBuilders{
	GitHub: ghforge.NewWithBase,
	GitLab: glforge.New,
	Gitea:  gitea.New,
}

// Repo figures out the forge, owner, and repo name from flags or the current
// git remote. The -R flag takes precedence; otherwise we read the "origin"
// remote URL and parse it.
func Repo(flagRepo, flagForgeType string) (forge forges.Forge, owner, repo, domain string, err error) {
	if flagRepo != "" {
		return repoFromFlag(flagRepo, flagForgeType)
	}
	return repoFromGitRemote(flagForgeType)
}

func repoFromFlag(flagRepo, flagForgeType string) (forges.Forge, string, string, string, error) {
	parts := strings.SplitN(flagRepo, "/", ownerRepoParts)
	if len(parts) != ownerRepoParts {
		return nil, "", "", "", fmt.Errorf("invalid repo format %q, expected OWNER/REPO", flagRepo)
	}
	owner, repo := parts[0], parts[1]

	domain := DomainFromForgeType(flagForgeType)
	client := newClient(domain)
	f, err := forgeForDomainMaybeConfig(context.Background(), client, domain)
	if err != nil {
		return nil, "", "", "", err
	}
	return f, owner, repo, domain, nil
}

func repoFromGitRemote(_ string) (forges.Forge, string, string, string, error) {
	domain, owner, repo, err := resolveRemote()
	if err != nil {
		return nil, "", "", "", err
	}

	client := newClient(domain)
	f, err := forgeForDomainMaybeConfig(context.Background(), client, domain)
	if err != nil {
		return nil, "", "", "", err
	}
	return f, owner, repo, domain, nil
}

func resolveRemote() (domain, owner, repo string, err error) {
	url, err := gitRemoteURL(remoteName)
	if err != nil {
		return "", "", "", fmt.Errorf("reading remote %q (not in a git repo, or remote not configured; use -R or --remote): %w", remoteName, err)
	}

	domain, owner, repo, err = forges.ParseRepoURL(url)
	if err != nil {
		return "", "", "", fmt.Errorf("parsing remote %q URL: %w", remoteName, err)
	}
	return mapSSHHost(domain), owner, repo, nil
}

// mapSSHHost translates a git-over-ssh hostname to the corresponding API
// hostname when the config declares them as different. Self-hosted GitLab
// can serve ssh on ssh.gitlab.test while the API lives at gitlab.test;
// without this mapping the parsed remote domain would point at the wrong host.
// Returns the input unchanged when no mapping is configured.
func mapSSHHost(domain string) string {
	cfg, err := config.Load()
	if err != nil {
		return domain
	}
	if api := cfg.DomainForSSHHost(domain); api != "" {
		return api
	}
	return domain
}

func gitRemoteURL(name string) (string, error) {
	out, err := exec.Command("git", "remote", "get-url", name).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func newClient(domain string) *forges.Client {
	token := TokenForDomain(domain)
	var opts []forges.Option
	if token != "" {
		opts = append(opts, forges.WithToken(domain, token))
	}

	// Register default forges first, so config-based registrations override them.
	hc := HTTPClient()
	defaults := map[string]forges.Forge{
		"github.com":    ghforge.New(TokenForDomain("github.com"), hc),
		"gitlab.com":    glforge.New("https://gitlab.com", TokenForDomain("gitlab.com"), hc),
		"codeberg.org":  gitea.New("https://codeberg.org", TokenForDomain("codeberg.org"), hc),
		"bitbucket.org": bitbucket.New(TokenForDomain("bitbucket.org"), hc),
	}
	for d, f := range defaults {
		opts = append(opts, forges.WithForge(d, f))
	}

	// If the config knows this domain's forge type, register it after defaults
	// so it takes precedence.
	if ft := configForgeType(domain); ft != "" {
		switch ft {
		case "gitea", "forgejo":
			opts = append(opts, forges.WithForge(domain, gitea.New("https://"+domain, token, hc)))
		case "gitlab":
			opts = append(opts, forges.WithForge(domain, glforge.New("https://"+domain, token, hc)))
		}
	}

	return forges.NewClient(opts...)
}

// forgeForDomainMaybeConfig tries the client's registered forges first. If that
// fails and the config declares a type for the domain, it registers the domain
// using that type (skipping network detection). Otherwise falls back to probing.
func forgeForDomainMaybeConfig(ctx context.Context, client *forges.Client, domain string) (forges.Forge, error) {
	f, err := client.ForgeFor(domain)
	if err == nil {
		return f, nil
	}
	token := TokenForDomain(domain)
	if regErr := client.RegisterDomain(ctx, domain, token, builders); regErr != nil {
		return nil, fmt.Errorf("unknown forge at %s: %w", domain, regErr)
	}
	return client.ForgeFor(domain)
}

// configForgeType returns the forge type for a domain from config files,
// or empty string if not configured.
func configForgeType(domain string) string {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.Domains[domain].Type
}

// TokenForDomain looks up an auth token. Checks environment variables first
// (highest precedence), then falls back to the user config file.
func TokenForDomain(domain string) string {
	if t := TokenForDomainEnv(domain); t != "" {
		return t
	}

	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.Domains[domain].Token
}

// TokenForDomainEnv looks up a token from environment variables only.
// Checks domain-specific variables first, then falls back to FORGE_TOKEN.
func TokenForDomainEnv(domain string) string {
	switch domain {
	case "github.com":
		if t := os.Getenv("GITHUB_TOKEN"); t != "" {
			return t
		}
		if t := os.Getenv("GH_TOKEN"); t != "" {
			return t
		}
	case "gitlab.com":
		if t := os.Getenv("GITLAB_TOKEN"); t != "" {
			return t
		}
		if t := os.Getenv("GLAB_TOKEN"); t != "" {
			return t
		}
	case "codeberg.org":
		if t := os.Getenv("GITEA_TOKEN"); t != "" {
			return t
		}
	case "bitbucket.org":
		if t := os.Getenv("BITBUCKET_TOKEN"); t != "" {
			return t
		}
	}

	// FORGE_TOKEN is a fallback for any domain without a specific token.
	return os.Getenv("FORGE_TOKEN")
}

// ForgeForDomain returns a Forge instance for the given domain.
// If the domain isn't a known forge, it checks config then probes the server.
func ForgeForDomain(domain string) (forges.Forge, error) {
	client := newClient(domain)
	return forgeForDomainMaybeConfig(context.Background(), client, domain)
}

// DomainFromForgeType returns the default domain for a forge type string.
// Checks FORGE_HOST first, then config default, then well-known defaults.
func DomainFromForgeType(forgeType string) string {
	if d := os.Getenv("FORGE_HOST"); d != "" {
		return d
	}

	// If no forge type given, check config for a default
	if forgeType == "" {
		cfg, err := config.Load()
		if err == nil && cfg != nil && cfg.Default.ForgeType != "" {
			forgeType = cfg.Default.ForgeType
		}
	}

	switch forgeType {
	case "gitlab":
		return "gitlab.com"
	case "gitea", "forgejo":
		return "codeberg.org"
	case "bitbucket":
		return "bitbucket.org"
	default:
		return "github.com"
	}
}

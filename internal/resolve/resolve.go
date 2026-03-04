package resolve

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/git-pkgs/forges"
	"github.com/git-pkgs/forges/internal/config"
)

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
	parts := strings.SplitN(flagRepo, "/", 2)
	if len(parts) != 2 {
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

func repoFromGitRemote(flagForgeType string) (forges.Forge, string, string, string, error) {
	url, err := gitRemoteURL("origin")
	if err != nil {
		return nil, "", "", "", fmt.Errorf("not in a git repo and -R not set: %w", err)
	}

	domain, owner, repo, err := forges.ParseRepoURL(url)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("parsing remote URL: %w", err)
	}

	client := newClient(domain)
	f, err := forgeForDomainMaybeConfig(context.Background(), client, domain)
	if err != nil {
		return nil, "", "", "", err
	}
	return f, owner, repo, domain, nil
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

	// If the config knows this domain's forge type, register it directly
	// so we skip auto-detection.
	if ft := configForgeType(domain); ft != "" {
		switch ft {
		case "gitea", "forgejo":
			opts = append(opts, forges.WithGitea(domain, token))
		case "gitlab":
			opts = append(opts, forges.WithGitLab(domain, token))
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
	if regErr := client.RegisterDomain(ctx, domain, token); regErr != nil {
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
// Checks FORGE_TOKEN first, then forge-specific variables.
func TokenForDomainEnv(domain string) string {
	if t := os.Getenv("FORGE_TOKEN"); t != "" {
		return t
	}

	switch domain {
	case "github.com":
		if t := os.Getenv("GITHUB_TOKEN"); t != "" {
			return t
		}
		return os.Getenv("GH_TOKEN")
	case "gitlab.com":
		if t := os.Getenv("GITLAB_TOKEN"); t != "" {
			return t
		}
		return os.Getenv("GLAB_TOKEN")
	case "codeberg.org":
		return os.Getenv("GITEA_TOKEN")
	case "bitbucket.org":
		return os.Getenv("BITBUCKET_TOKEN")
	}

	return ""
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

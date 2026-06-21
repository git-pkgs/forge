package resolve

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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

var (
	remoteName        = "origin"
	hostOverride      string
	forgeTypeOverride string

	// testForge allows tests to inject a mock forge. When set, Repo() returns
	// this forge directly without network or git resolution.
	testForge  forges.Forge
	testOwner  string
	testRepo   string
	testDomain string
)

// SetRemote sets which git remote to read when resolving the current
// repository. The CLI calls this from the --remote persistent flag.
// An empty string is ignored so callers can pass the flag value
// unconditionally.
func SetRemote(name string) {
	if name != "" {
		remoteName = name
	}
}

// RemoteName returns the name of the git remote being used for resolution.
// This is "origin" by default, or whatever was set via SetRemote.
func RemoteName() string {
	return remoteName
}

// SetHost forces a specific forge domain, taking precedence over FORGE_HOST,
// --forge-type, and git remote detection. The CLI calls this from the --host
// persistent flag. An empty string is ignored.
func SetHost(host string) {
	if host != "" {
		hostOverride = host
	}
}

// SetForgeType forces the API client implementation for the resolved domain,
// skipping config lookup and network probing. The CLI calls this from the
// --forge-type persistent flag. An empty string is ignored.
func SetForgeType(forgeType string) {
	if forgeType != "" {
		forgeTypeOverride = forgeType
	}
}

// SetTestForge configures a mock forge for testing. When set, Repo() returns
// this forge directly without network or git resolution.
func SetTestForge(forge forges.Forge, owner, repo, domain string) {
	testForge = forge
	testOwner = owner
	testRepo = repo
	testDomain = domain
}

// ResetTestForge clears the test forge configuration.
func ResetTestForge() {
	testForge = nil
	testOwner = ""
	testRepo = ""
	testDomain = ""
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
	if testForge != nil {
		return testForge, testOwner, testRepo, testDomain, nil
	}
	if flagRepo != "" {
		return repoFromFlag(flagRepo, flagForgeType)
	}
	return repoFromGitRemote(flagForgeType)
}

func repoFromFlag(flagRepo, flagForgeType string) (forges.Forge, string, string, string, error) {
	parts := strings.Split(flagRepo, "/")

	var domain, owner, repo string
	switch len(parts) {
	case 2:
		// owner/repo
		owner, repo = parts[0], parts[1]
		domain = Domain(flagForgeType)
	case 3:
		// host/owner/repo
		domain, owner, repo = parts[0], parts[1], parts[2]
	default:
		return nil, "", "", "", fmt.Errorf("invalid repo format %q, expected OWNER/REPO or HOST/OWNER/REPO", flagRepo)
	}

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

// ResourceFromURL resolves a forge and resource details from a full forge URL.
func ResourceFromURL(rawURL string) (forge forges.Forge, domain string, ref *forges.ResourceRef, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme == "" {
		u, err = url.Parse("https://" + rawURL)
		if err != nil {
			return nil, "", nil, fmt.Errorf("invalid URL: %w", err)
		}
	}

	domain = u.Hostname()
	path := strings.Trim(u.Path, "/")

	var f forges.Forge
	if testForge != nil {
		f = testForge
	} else {
		client := newClient(domain)
		f, err = forgeForDomainMaybeConfig(context.Background(), client, domain)
		if err != nil {
			return nil, "", nil, err
		}
	}

	parts := strings.Split(path, "/")
	ref, err = f.ParsePath(parts)
	if err != nil {
		return nil, "", nil, err
	}

	return f, domain, ref, nil
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

// OwnerForBranch returns the repository owner for the remote that the given
// branch tracks. This is useful for determining which fork a branch was pushed
// to when creating pull requests.
func OwnerForBranch(ctx context.Context, branch string) (string, error) {
	remoteKey := fmt.Sprintf("branch.%s.remote", branch)
	out, err := exec.CommandContext(ctx, "git", "config", "--get", remoteKey).Output()
	if err != nil {
		return "", err
	}
	remote := strings.TrimSpace(string(out))
	if remote == "" {
		return "", fmt.Errorf("no remote configured")
	}

	remoteURL, err := gitRemoteURL(remote)
	if err != nil {
		return "", err
	}

	_, owner, _, err := forges.ParseRepoURL(remoteURL)
	if err != nil {
		return "", err
	}
	return owner, nil
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

	// If --forge-type was given or the config knows this domain's type,
	// register it after defaults so it takes precedence and probing is skipped.
	ft := forgeTypeOverride
	if ft == "" {
		ft = configForgeType(domain)
	}
	if f := forgeForType(ft, "https://"+domain, token, hc); f != nil {
		opts = append(opts, forges.WithForge(domain, f))
	}

	return forges.NewClient(opts...)
}

func forgeForType(forgeType, baseURL, token string, hc *http.Client) forges.Forge {
	switch forgeType {
	case "gitea", "forgejo":
		return gitea.New(baseURL, token, hc)
	case "gitlab":
		return glforge.New(baseURL, token, hc)
	case "github":
		return ghforge.NewWithBase(baseURL, token, hc)
	}
	return nil
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
		return nil, fmt.Errorf("unknown forge at %s: %w (use --forge-type, or set type under [%s] in config, to skip detection)", domain, regErr, domain)
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
		if t := os.Getenv("FORGEJO_TOKEN"); t != "" {
			return t
		}
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

// Domain decides which forge host to talk to when the user supplies a bare
// owner or owner/repo argument. Precedence: --host flag, FORGE_HOST env,
// explicit --forge-type, the current directory's git remote, the config
// default forge type, then github.com.
func Domain(forgeType string) string {
	if hostOverride != "" {
		return hostOverride
	}
	if d := os.Getenv("FORGE_HOST"); d != "" {
		return d
	}
	if forgeType != "" {
		return defaultDomainForType(forgeType)
	}
	if d, _, _, err := resolveRemote(); err == nil {
		return d
	}
	if cfg, err := config.Load(); err == nil && cfg != nil && cfg.Default.ForgeType != "" {
		return defaultDomainForType(cfg.Default.ForgeType)
	}
	return "github.com"
}

func defaultDomainForType(forgeType string) string {
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

package forges

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/git-pkgs/purl"
)

// ErrNotFound is returned when the requested resource does not exist.
var ErrNotFound = errors.New("not found")

// ErrOwnerNotFound is returned when the requested owner (org or user) does not exist.
var ErrOwnerNotFound = errors.New("owner not found")

// ErrNotSupported is returned when a forge does not support an operation.
var ErrNotSupported = errors.New("not supported by this forge")

// HTTPError represents a non-OK HTTP response from a forge API.
type HTTPError struct {
	StatusCode int
	URL        string
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("forge: HTTP %d from %s", e.StatusCode, e.URL)
}

// Forge is the interface each forge backend implements.
type Forge interface {
	Repos() RepoService
	Issues() IssueService
	PullRequests() PullRequestService
	Labels() LabelService
	Milestones() MilestoneService
	Releases() ReleaseService
	CI() CIService
	Branches() BranchService
	DeployKeys() DeployKeyService
	Secrets() SecretService
	Notifications() NotificationService
	Reviews() ReviewService
	Files() FileService
	Collaborators() CollaboratorService
	GetRateLimit(ctx context.Context) (*RateLimit, error)
}

// Client routes requests to the appropriate Forge based on the URL domain.
type Client struct {
	forges     map[string]Forge
	tokens     map[string]string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithToken sets the API token for the given domain.
func WithToken(domain, token string) Option {
	return func(c *Client) {
		c.tokens[domain] = token
	}
}

// WithHTTPClient overrides the default HTTP client used by forge backends.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithForge registers a Forge implementation for the given domain.
func WithForge(domain string, f Forge) Option {
	return func(c *Client) {
		c.forges[domain] = f
	}
}

// NewClient creates a Client and applies the given options.
func NewClient(opts ...Option) *Client {
	c := &Client{
		forges: make(map[string]Forge),
		tokens: make(map[string]string),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Tokens returns the token map so callers can read tokens set via WithToken.
func (c *Client) Tokens() map[string]string {
	return c.tokens
}

// HTTPClient returns the HTTP client configured on this Client, or nil.
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// RegisterDomain detects the forge type for a domain and registers the
// appropriate Forge using the provided builder functions.
func (c *Client) RegisterDomain(ctx context.Context, domain, token string, builders ForgeBuilders) error {
	ft, err := DetectForgeType(ctx, domain, c.httpClient)
	if err != nil {
		return fmt.Errorf("detecting forge type for %s: %w", domain, err)
	}
	c.tokens[domain] = token
	baseURL := "https://" + domain
	switch ft {
	case GitHub:
		c.forges[domain] = builders.GitHub(baseURL, token, c.httpClient)
	case GitLab:
		c.forges[domain] = builders.GitLab(baseURL, token, c.httpClient)
	case Gitea, Forgejo:
		c.forges[domain] = builders.Gitea(baseURL, token, c.httpClient)
	default:
		return fmt.Errorf("unsupported forge type %q for %s", ft, domain)
	}
	return nil
}

// ForgeBuilders holds constructor functions for each forge type.
// Used by RegisterDomain to create the right forge after detection.
type ForgeBuilders struct {
	GitHub func(baseURL, token string, hc *http.Client) Forge
	GitLab func(baseURL, token string, hc *http.Client) Forge
	Gitea  func(baseURL, token string, hc *http.Client) Forge
}

func (c *Client) forgeFor(domain string) (Forge, error) {
	f, ok := c.forges[domain]
	if !ok {
		return nil, fmt.Errorf("no forge registered for domain %q", domain)
	}
	return f, nil
}

// ForgeFor returns the Forge implementation registered for the given domain.
func (c *Client) ForgeFor(domain string) (Forge, error) {
	return c.forgeFor(domain)
}

// FetchRepository fetches normalized repository metadata from a URL string.
func (c *Client) FetchRepository(ctx context.Context, repoURL string) (*Repository, error) {
	domain, owner, repo, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}
	f, err := c.forgeFor(domain)
	if err != nil {
		return nil, err
	}
	return f.Repos().Get(ctx, owner, repo)
}

// FetchRepositoryFromPURL fetches repository metadata using a PURL's
// repository_url qualifier.
func (c *Client) FetchRepositoryFromPURL(ctx context.Context, p *purl.PURL) (*Repository, error) {
	repoURL := p.RepositoryURL()
	if repoURL == "" {
		return nil, fmt.Errorf("PURL has no repository_url qualifier")
	}
	return c.FetchRepository(ctx, repoURL)
}

// FetchTags fetches git tags from a URL string.
func (c *Client) FetchTags(ctx context.Context, repoURL string) ([]Tag, error) {
	domain, owner, repo, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}
	f, err := c.forgeFor(domain)
	if err != nil {
		return nil, err
	}
	return f.Repos().ListTags(ctx, owner, repo)
}

// ListRepositories lists all repositories for an owner on the given domain.
func (c *Client) ListRepositories(ctx context.Context, domain, owner string, opts ListRepoOpts) ([]Repository, error) {
	f, err := c.forgeFor(domain)
	if err != nil {
		return nil, err
	}
	return f.Repos().List(ctx, owner, opts)
}

// FilterRepos applies archived and fork filters to a slice of repositories.
func FilterRepos(repos []Repository, opts ListRepoOpts) []Repository {
	n := 0
	for _, r := range repos {
		switch opts.Archived {
		case ArchivedExclude:
			if r.Archived {
				continue
			}
		case ArchivedOnly:
			if !r.Archived {
				continue
			}
		}
		switch opts.Forks {
		case ForkExclude:
			if r.Fork {
				continue
			}
		case ForkOnly:
			if !r.Fork {
				continue
			}
		}
		repos[n] = r
		n++
	}
	return repos[:n]
}

// FetchTagsFromPURL fetches git tags using a PURL's repository_url qualifier.
func (c *Client) FetchTagsFromPURL(ctx context.Context, p *purl.PURL) ([]Tag, error) {
	repoURL := p.RepositoryURL()
	if repoURL == "" {
		return nil, fmt.Errorf("PURL has no repository_url qualifier")
	}
	return c.FetchTags(ctx, repoURL)
}

// ParseRepoURL extracts the domain, owner, and repo from a repository URL.
// It handles https://, schemeless, and git@host:owner/repo SSH URLs, and
// strips .git suffixes and extra path segments.
func ParseRepoURL(rawURL string) (domain, owner, repo string, err error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", "", "", fmt.Errorf("empty URL")
	}

	// Handle git@ SSH URLs: git@github.com:owner/repo.git
	if strings.HasPrefix(rawURL, "git@") {
		rawURL = strings.TrimPrefix(rawURL, "git@")
		colonIdx := strings.Index(rawURL, ":")
		if colonIdx < 0 {
			return "", "", "", fmt.Errorf("invalid SSH URL: missing colon")
		}
		domain = rawURL[:colonIdx]
		path := rawURL[colonIdx+1:]
		return splitOwnerRepo(domain, path)
	}

	// Add scheme if missing
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid URL: %w", err)
	}
	domain = u.Hostname()
	return splitOwnerRepo(domain, u.Path)
}

func splitOwnerRepo(domain, path string) (string, string, string, error) {
	path = strings.TrimSuffix(path, ".git")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("URL path must contain owner/repo, got %q", path)
	}
	return domain, parts[0], parts[1], nil
}

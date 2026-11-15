// Package factory builds forges.Client values wired up with every provider, so
// callers can pass any repository URL without choosing a backend themselves.
package factory

import (
	"net/http"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/bitbucket"
	"github.com/git-pkgs/forge/gerrit"
	"github.com/git-pkgs/forge/gitea"
	"github.com/git-pkgs/forge/github"
	"github.com/git-pkgs/forge/gitlab"
	"github.com/git-pkgs/forge/radicle"
	"github.com/git-pkgs/forge/tangled"
)

// Builders returns the constructors for every provider that can be detected on
// a self-hosted domain.
func Builders() forges.ForgeBuilders {
	return forges.ForgeBuilders{
		GitHub:  github.NewWithBase,
		GitLab:  gitlab.New,
		Gitea:   gitea.New,
		Gerrit:  gerrit.New,
		Tangled: tangled.New,
		Radicle: radicle.New,
	}
}

// DefaultForges returns a Forge for each well-known public domain, keyed by
// domain. The token function supplies the token for each domain and may return
// an empty string for anonymous access.
func DefaultForges(token func(domain string) string, hc *http.Client) map[string]forges.Forge {
	return map[string]forges.Forge{
		"github.com":    github.New(token("github.com"), hc),
		"gitlab.com":    gitlab.New("https://gitlab.com", token("gitlab.com"), hc),
		"codeberg.org":  gitea.New("https://codeberg.org", token("codeberg.org"), hc),
		"bitbucket.org": bitbucket.New(token("bitbucket.org"), hc),
		"tangled.org":   tangled.New("https://tangled.org", token("tangled.org"), hc),
		// Upstream serves GitHub's REST API at /api/v3.
		"upstream.codes": github.NewWithBase("https://upstream.codes", token("upstream.codes"), hc),
	}
}

// DefaultHostAliases returns the git hosts of well-known forges that serve git
// from a different host than their API, mapped to the API's domain.
func DefaultHostAliases() map[string]string {
	return map[string]string{
		"git.coline.app":     "upstream.codes",
		"ssh.git.coline.app": "upstream.codes",
	}
}

// NewClient returns a Client with the default forges and host aliases
// registered and detection enabled for every other domain. Options apply
// first, so WithToken and WithHTTPClient reach the default forges, and a
// domain registered with WithForge or WithHostAlias keeps that registration.
func NewClient(opts ...forges.Option) *forges.Client {
	base := []forges.Option{forges.WithDetection(Builders())}
	for host, domain := range DefaultHostAliases() {
		base = append(base, forges.WithHostAlias(host, domain))
	}
	c := forges.NewClient(append(base, opts...)...)
	tokens := c.Tokens()
	defaults := DefaultForges(func(domain string) string { return tokens[domain] }, c.HTTPClient())
	for domain, f := range defaults {
		if _, err := c.ForgeFor(domain); err != nil {
			forges.WithForge(domain, f)(c)
		}
	}
	return c
}

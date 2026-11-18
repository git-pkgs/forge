package radicle

import (
	"fmt"
	"strings"

	forges "github.com/git-pkgs/forge"
)

// ParsePath implements Forge.ParsePath for Radicle URLs. It accepts any path
// that contains a repository ID segment, such as /rad:z3gq.../tree or
// /nodes/seed.radicle.dev/rad:z3gq.../issues/1e52373d. Radicle issue and
// patch IDs are not numeric, so only the repository is extracted.
func (f *radicleForge) ParsePath(parts []string) (*forges.ResourceRef, error) {
	for _, p := range parts {
		if strings.HasPrefix(p, RIDPrefix) {
			id, err := rid("", p)
			if err != nil {
				return nil, err
			}
			return &forges.ResourceRef{Repo: id}, nil
		}
	}
	return nil, fmt.Errorf("URL path must contain a radicle repository ID (rad:...)")
}

// RemoteScheme is the URL scheme of git remotes set up by rad clone and rad init.
const RemoteScheme = "rad://"

// ParseRemoteURL returns the repository ID from a rad://<RID> fetch URL or a
// rad://<RID>/<NID> push URL.
func ParseRemoteURL(rawURL string) (string, error) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(rawURL), RemoteScheme)
	if !ok {
		return "", fmt.Errorf("not a radicle remote URL: %q", rawURL)
	}
	id, _, _ := strings.Cut(rest, "/")
	if strings.HasPrefix(id, RIDPrefix) {
		return "", fmt.Errorf("invalid radicle remote URL %q", rawURL)
	}
	return rid("", id)
}

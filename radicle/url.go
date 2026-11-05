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

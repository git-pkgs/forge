package gerrit

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	forge "github.com/git-pkgs/forge"
)

// ParsePath implements Forge.ParsePath for Gerrit URLs.
func (f *gerritForge) ParsePath(parts []string) (*forge.ResourceRef, error) {
	if len(parts) == 0 {
		return nil, fmt.Errorf("URL path must contain a Gerrit resource")
	}

	if parts[0] == "c" {
		for i, part := range parts {
			if part != "+" || i+1 >= len(parts) {
				continue
			}
			number, err := strconv.Atoi(parts[i+1])
			if err != nil {
				return nil, fmt.Errorf("invalid change number %q", parts[i+1])
			}
			project := strings.Join(parts[1:i], "/")
			decoded, err := url.PathUnescape(project)
			if err != nil {
				return nil, err
			}
			owner, repo := splitProjectName(decoded)
			return &forge.ResourceRef{
				Owner:  owner,
				Repo:   repo,
				Type:   forge.ResourceTypePR,
				Number: number,
			}, nil
		}
	}

	if len(parts) >= 3 && parts[0] == "admin" && parts[1] == "repos" {
		project, err := url.PathUnescape(parts[2])
		if err != nil {
			return nil, err
		}
		owner, repo := splitProjectName(project)
		return &forge.ResourceRef{Owner: owner, Repo: repo}, nil
	}

	return nil, fmt.Errorf("unsupported Gerrit URL path")
}

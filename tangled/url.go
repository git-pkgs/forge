package tangled

import (
	"fmt"
	"strconv"

	forges "github.com/git-pkgs/forge"
)

// ParsePath implements Forge.ParsePath for Tangled URLs.
func (f *tangledForge) ParsePath(parts []string) (*forges.ResourceRef, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("URL path must contain owner/repo")
	}

	ref := &forges.ResourceRef{
		Owner: parts[0],
		Repo:  parts[1],
	}

	if len(parts) >= 4 {
		num, err := strconv.Atoi(parts[3])
		if err != nil {
			return nil, fmt.Errorf("invalid number %q", parts[3])
		}
		ref.Number = num

		switch parts[2] {
		case "pulls", "pull":
			ref.Type = forges.ResourceTypePR
		case "issues":
			ref.Type = forges.ResourceTypeIssue
		}
	}

	return ref, nil
}

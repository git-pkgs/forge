package gitlab

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/git-pkgs/forge"
)

// ParsePath implements Forge.ParsePath for GitLab URLs.
func (f *gitLabForge) ParsePath(parts []string) (*forges.ResourceRef, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("URL path must contain owner/repo")
	}

	// Find /-/ separator marking resource paths
	dashIdx := -1
	for i, part := range parts {
		if part == "-" {
			dashIdx = i
			break
		}
	}

	ref := &forges.ResourceRef{}

	if dashIdx == -1 {
		ref.Owner = strings.Join(parts[:len(parts)-1], "/")
		ref.Repo = parts[len(parts)-1]
		return ref, nil
	}

	if dashIdx < 2 {
		return nil, fmt.Errorf("URL path must contain owner/repo before /-/")
	}
	ref.Owner = strings.Join(parts[:dashIdx-1], "/")
	ref.Repo = parts[dashIdx-1]

	if dashIdx+2 < len(parts) {
		num, err := strconv.Atoi(parts[dashIdx+2])
		if err != nil {
			return nil, fmt.Errorf("invalid number %q", parts[dashIdx+2])
		}
		ref.Number = num

		switch parts[dashIdx+1] {
		case "merge_requests":
			ref.Type = forges.ResourceTypePR
		case "issues":
			ref.Type = forges.ResourceTypeIssue
		}
	}

	return ref, nil
}

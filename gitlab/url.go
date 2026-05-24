package gitlab

import (
	"fmt"
	"strconv"
	"strings"
)

// ParsePath implements Forge.ParsePath for GitLab URLs.
func (f *gitLabForge) ParsePath(parts []string) (owner, repo, resourceType string, number int, err error) {
	return parsePath(parts)
}

// parsePath parses GitLab URL path segments into resource components.
// Formats: group/project, group/subgroup/project, group/project/-/merge_requests/123
func parsePath(parts []string) (owner, repo, resourceType string, number int, err error) {
	if len(parts) < 2 {
		return "", "", "", 0, fmt.Errorf("URL path must contain owner/repo")
	}

	// Find /-/ separator marking resource paths
	dashIdx := -1
	for i, part := range parts {
		if part == "-" {
			dashIdx = i
			break
		}
	}

	if dashIdx == -1 {
		owner = strings.Join(parts[:len(parts)-1], "/")
		repo = parts[len(parts)-1]
		return owner, repo, "", 0, nil
	}

	if dashIdx < 2 {
		return "", "", "", 0, fmt.Errorf("URL path must contain owner/repo before /-/")
	}
	owner = strings.Join(parts[:dashIdx-1], "/")
	repo = parts[dashIdx-1]

	if dashIdx+2 < len(parts) {
		switch parts[dashIdx+1] {
		case "merge_requests":
			resourceType = "pr"
			number, err = strconv.Atoi(parts[dashIdx+2])
			if err != nil {
				return "", "", "", 0, fmt.Errorf("invalid MR number %q", parts[dashIdx+2])
			}
		case "issues":
			resourceType = "issue"
			number, err = strconv.Atoi(parts[dashIdx+2])
			if err != nil {
				return "", "", "", 0, fmt.Errorf("invalid issue number %q", parts[dashIdx+2])
			}
		}
	}

	return owner, repo, resourceType, number, nil
}

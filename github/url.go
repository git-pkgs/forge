package github

import (
	"fmt"
	"strconv"
)

// ParsePath implements Forge.ParsePath for GitHub URLs.
func (f *gitHubForge) ParsePath(parts []string) (owner, repo, resourceType string, number int, err error) {
	return parsePath(parts)
}

// parsePath parses GitHub URL path segments into resource components.
// Formats: owner/repo, owner/repo/pull/123, owner/repo/issues/456
func parsePath(parts []string) (owner, repo, resourceType string, number int, err error) {
	if len(parts) < 2 {
		return "", "", "", 0, fmt.Errorf("URL path must contain owner/repo")
	}

	owner, repo = parts[0], parts[1]

	if len(parts) >= 4 {
		switch parts[2] {
		case "pull":
			resourceType = "pr"
			number, err = strconv.Atoi(parts[3])
			if err != nil {
				return "", "", "", 0, fmt.Errorf("invalid PR number %q", parts[3])
			}
		case "issues":
			resourceType = "issue"
			number, err = strconv.Atoi(parts[3])
			if err != nil {
				return "", "", "", 0, fmt.Errorf("invalid issue number %q", parts[3])
			}
		}
	}

	return owner, repo, resourceType, number, nil
}

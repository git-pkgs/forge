package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/git-pkgs/forge"
)

// GetOrFetchBaseBranch returns the base branch of the given branch.
// It first checks the local git configuration for a cached value.
// If not found, it queries the forge API for an open pull request for the branch,
// caches the base branch name in the local git configuration, and returns it.
// If branch is empty, it uses the current branch.
func GetOrFetchBaseBranch(ctx context.Context, f forges.Forge, owner, repo, branch string, forceRefresh bool) (string, error) {
	if branch == "" {
		curr, err := runGit(ctx, "branch", "--show-current")
		if err != nil {
			return "", fmt.Errorf("failed to get current branch: %w", err)
		}
		branch = curr
	}
	if branch == "" {
		return "", fmt.Errorf("empty branch name")
	}

	// 1. Check local git config
	configKey := fmt.Sprintf("branch.%s.forge-merge-base", branch)
	if !forceRefresh {
		if cached, err := runGit(ctx, "config", "--get", configKey); err == nil && cached != "" {
			return cached, nil
		}
	}

	// 2. Fetch base branch via forge API
	prs, err := f.PullRequests().List(ctx, owner, repo, forges.ListPROpts{
		State: "open",
		Head:  branch,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list pull requests: %w", err)
	}

	var baseBranch string
	for _, pr := range prs {
		if pr.Head.Ref == branch || strings.HasSuffix(pr.Head.Ref, ":"+branch) {
			baseBranch = pr.Base.Ref
			break
		}
	}

	if baseBranch == "" {
		return "", fmt.Errorf("no open pull request found for branch %q", branch)
	}

	// 3. Cache the resolved base branch in local git config
	// Even if caching fails, we still return the resolved base branch.
	_, _ = runGit(ctx, "config", "--local", configKey, baseBranch)

	return baseBranch, nil
}

func runGit(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

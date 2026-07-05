// Package git provides helpers for interacting with local git repositories and configurations.
package git

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/git-pkgs/forge"
)

const (
	forgeMergeBaseKey = "forge-merge-base"
	forgePRKey        = "forge-pr"
)

var prRefRE = regexp.MustCompile(`^refs/pull/(\d+)/head$`)

// GetOrFetchBaseBranch returns the base branch of the given branch.
// It first checks the local git configuration for a cached value.
// If not found, it queries the forge API for an open pull request for the branch,
// caches the base branch name in the local git configuration, and returns it.
// If branch is empty, it uses the current branch.
func GetOrFetchBaseBranch(ctx context.Context, f forges.Forge, dir, owner, repo, branch string, forceRefresh bool) (string, error) {
	if branch == "" {
		curr, err := CurrentBranch(ctx, dir)
		if err != nil {
			return "", err
		}
		branch = curr
	}
	if branch == "" {
		return "", fmt.Errorf("empty branch name")
	}

	// 1. Check local git config
	configKey := branchConfigKey(branch, forgeMergeBaseKey)
	if !forceRefresh {
		if cached, err := getLocalConfig(ctx, dir, configKey); err == nil && cached != "" {
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
		// GitLab, Gitea, and Bitbucket do not support filtering by Head on the server side,
		// so we must filter client-side. We check if the head branch ref matches the requested branch.
		if pr.Head.Ref == branch {
			baseBranch = pr.Base.Ref
			break
		}
	}

	if baseBranch == "" {
		return "", fmt.Errorf("no open pull request found for branch %q", branch)
	}

	// 3. Cache the resolved base branch in local git config
	// Even if caching fails, we still return the resolved base branch.
	_ = SetBaseBranch(ctx, dir, branch, baseBranch)

	return baseBranch, nil
}

// SetBaseBranch caches the base branch for a branch in the local git configuration.
func SetBaseBranch(ctx context.Context, dir, branch, base string) error {
	if branch == "" {
		return fmt.Errorf("empty branch name")
	}
	if base == "" {
		return fmt.Errorf("empty base branch name")
	}
	configKey := branchConfigKey(branch, forgeMergeBaseKey)
	_, err := runGit(ctx, dir, "config", "--local", configKey, base)
	return err
}

// SetPRNumber caches the pull request number for a branch in the local git configuration.
func SetPRNumber(ctx context.Context, dir, branch string, number int) error {
	if branch == "" {
		return fmt.Errorf("empty branch name")
	}
	if number <= 0 {
		return fmt.Errorf("invalid pull request number %d", number)
	}
	_, err := runGit(ctx, dir, "config", "--local", branchConfigKey(branch, forgePRKey), strconv.Itoa(number))
	return err
}

// GetPRNumber returns the cached pull request number for a branch.
func GetPRNumber(ctx context.Context, dir, branch string) (int, error) {
	if branch == "" {
		return 0, fmt.Errorf("empty branch name")
	}
	if out, err := getLocalConfig(ctx, dir, branchConfigKey(branch, forgePRKey)); err == nil {
		return strconv.Atoi(out)
	}

	// Fall back to gh CLI's format (refs/pull/<n>/head in branch.<name>.merge).
	// The regex only matches refs/pull/<n>/head, so refs/heads/* values are
	// safely rejected.
	out, err := getLocalConfig(ctx, dir, branchConfigKey(branch, "merge"))
	if err != nil {
		return 0, err
	}
	m := prRefRE.FindStringSubmatch(out)
	if m == nil {
		return 0, fmt.Errorf("not a PR ref")
	}
	return strconv.Atoi(m[1])
}

// CurrentBranch returns the current local branch name.
func CurrentBranch(ctx context.Context, dir string) (string, error) {
	branch, err := runGit(ctx, dir, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return branch, nil
}

func branchConfigKey(branch, name string) string {
	return fmt.Sprintf("branch.%s.%s", branch, name)
}

func getLocalConfig(ctx context.Context, dir, key string) (string, error) {
	return runGit(ctx, dir, "config", "--local", "--get", key)
}

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
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

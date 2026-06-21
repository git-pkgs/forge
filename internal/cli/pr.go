package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

const (
	maxPRTitleLength = 60
	defaultPRLimit   = 30
)

var prCmd = &cobra.Command{
	Use:     "pr",
	Aliases: []string{"mr"},
	Short:   "Manage pull requests",
}

func init() {
	rootCmd.AddCommand(prCmd)
	prCmd.AddCommand(prViewCmd())
	prCmd.AddCommand(prListCmd())
	prCmd.AddCommand(prCreateCmd())
	prCmd.AddCommand(prCloseCmd())
	prCmd.AddCommand(prReopenCmd())
	prCmd.AddCommand(prEditCmd())
	prCmd.AddCommand(prMergeCmd())
	prCmd.AddCommand(prDiffCmd())
	prCmd.AddCommand(prCommentCmd())
	prCmd.AddCommand(prReactionsCmd())
	prCmd.AddCommand(prReactCmd())
	prCmd.AddCommand(prCheckoutCmd())
}

func prViewCmd() *cobra.Command {
	var (
		flagComments bool
		flagWeb      bool
		flagJSON     string
	)

	cmd := &cobra.Command{
		Use:   "view [<number>]",
		Short: "View a pull request",
		Long: `View a pull request by number, or view the PR for the current branch.

If no number is given, finds and displays the pull request whose head
branch matches the current git branch.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("json") {
				var hints []string
				hints = append(hints, "forge --output json pr view <number>")
				if strings.Contains(flagJSON, "comments") {
					hints = append(hints, "forge pr view --comments <number>")
				}
				if strings.Contains(flagJSON, "reviews") {
					hints = append(hints, "forge pr review list <number>")
				}
				return fmt.Errorf("--json is not supported; use --output json instead (field selection is not supported)\n\nTry: %s", strings.Join(hints, "\n     "))
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			var number int
			if len(args) > 0 {
				number, err = strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("invalid PR number: %s", args[0])
				}
			} else {
				number, err = findPRForCurrentBranch(cmd.Context(), forge, owner, repoName)
				if err != nil {
					return err
				}
			}

			pr, err := forge.PullRequests().Get(cmd.Context(), owner, repoName, number)
			if err != nil {
				return fmt.Errorf("getting PR #%d: %w", number, err)
			}

			if flagWeb {
				return openBrowser(pr.HTMLURL)
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(pr)
			}

			printPRDetails(pr)

			if flagComments {
				comments, err := forge.PullRequests().ListComments(cmd.Context(), owner, repoName, number)
				if err != nil {
					return err
				}
				for _, c := range comments {
					_, _ = fmt.Fprintln(os.Stdout)
					_, _ = fmt.Fprintf(os.Stdout, "--- %s ---\n", output.Sanitize(c.Author.Login))
					_, _ = fmt.Fprintln(os.Stdout, output.Sanitize(c.Body))
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagComments, "comments", "c", false, "Show comments")
	cmd.Flags().BoolVarP(&flagWeb, "web", "w", false, "Open in browser")
	cmd.Flags().StringVar(&flagJSON, "json", "", "Not supported; use --output json")
	cmd.Flags().Lookup("json").NoOptDefVal = " "
	_ = cmd.Flags().MarkHidden("json")
	return cmd
}

func printPRDetails(pr *forges.PullRequest) {
	_, _ = fmt.Fprintf(os.Stdout, "#%d %s\n", pr.Number, output.Sanitize(pr.Title))
	_, _ = fmt.Fprintf(os.Stdout, "State:   %s\n", pr.State)
	_, _ = fmt.Fprintf(os.Stdout, "Author:  %s\n", output.Sanitize(pr.Author.Login))
	_, _ = fmt.Fprintf(os.Stdout, "Branch:  %s -> %s\n", pr.Head.Ref, pr.Base.Ref)

	if pr.Draft {
		_, _ = fmt.Fprintln(os.Stdout, "Draft:   yes")
	}

	if len(pr.Reviewers) > 0 {
		names := make([]string, len(pr.Reviewers))
		for i, r := range pr.Reviewers {
			names[i] = output.Sanitize(r.Login)
		}
		_, _ = fmt.Fprintf(os.Stdout, "Review:  %s\n", strings.Join(names, ", "))
	}

	if len(pr.Labels) > 0 {
		names := make([]string, len(pr.Labels))
		for i, l := range pr.Labels {
			names[i] = output.Sanitize(l.Name)
		}
		_, _ = fmt.Fprintf(os.Stdout, "Labels:  %s\n", strings.Join(names, ", "))
	}

	if pr.Milestone != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Mile:    %s\n", output.Sanitize(pr.Milestone.Title))
	}

	if pr.Additions > 0 || pr.Deletions > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Changes: +%d -%d (%d files)\n", pr.Additions, pr.Deletions, pr.ChangedFiles)
	}

	if pr.Body != "" {
		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, output.Sanitize(pr.Body))
	}
}

func prListCmd() *cobra.Command {
	var (
		flagState  string
		flagAuthor string
		flagHead   string
		flagBase   string
		flagLabels []string
		flagLimit  int
		flagSort   string
		flagOrder  string
		flagWeb    bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List pull requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if flagWeb {
				repo, err := forge.Repos().Get(cmd.Context(), owner, repoName)
				if err != nil {
					return fmt.Errorf("getting repository: %w", err)
				}
				return openBrowser(forge.PullRequests().ListURL(repo.HTMLURL))
			}

			opts := forges.ListPROpts{
				State:  flagState,
				Author: flagAuthor,
				Head:   flagHead,
				Base:   flagBase,
				Labels: flagLabels,
				Sort:   flagSort,
				Order:  flagOrder,
				Limit:  flagLimit,
			}

			prs, err := forge.PullRequests().List(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return fmt.Errorf("listing pull requests: %w", err)
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(prs)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(prs))
				for i, pr := range prs {
					lines[i] = fmt.Sprintf("%d\t%s", pr.Number, output.Sanitize(pr.Title))
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"#", "TITLE", "AUTHOR", "HEAD", "UPDATED"}
			rows := make([][]string, len(prs))
			for i, pr := range prs {
				title := output.Sanitize(pr.Title)
				if len(title) > maxPRTitleLength {
					title = title[:maxPRTitleLength-3] + "..."
				}
				rows[i] = []string{
					strconv.Itoa(pr.Number),
					title,
					output.Sanitize(pr.Author.Login),
					pr.Head.Ref,
					pr.UpdatedAt.Format("2006-01-02"),
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagState, "state", "s", "open", "Filter by state: open, closed, merged, all")
	cmd.Flags().StringVarP(&flagAuthor, "author", "A", "", "Filter by author")
	cmd.Flags().StringVar(&flagHead, "head", "", "Filter by head branch")
	cmd.Flags().StringVar(&flagBase, "base", "", "Filter by base branch")
	cmd.Flags().StringSliceVarP(&flagLabels, "label", "l", nil, "Filter by label")
	cmd.Flags().StringSliceVar(&flagLabels, "labels", nil, "Filter by label")
	_ = cmd.Flags().MarkHidden("labels")
	cmd.Flags().IntVarP(&flagLimit, "limit", "L", defaultPRLimit, "Maximum number of PRs")
	cmd.Flags().StringVar(&flagSort, "sort", "", "Sort by: created, updated")
	cmd.Flags().StringVar(&flagOrder, "order", "", "Sort order: asc, desc")
	cmd.Flags().BoolVarP(&flagWeb, "web", "w", false, "Open in browser")
	return cmd
}

func prCreateCmd() *cobra.Command {
	var (
		flagTitle     string
		flagBody      string
		flagHead      string
		flagBase      string
		flagDraft     bool
		flagReviewers []string
		flagAssignees []string
		flagLabels    []string
		flagMilestone string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"new"},
		Short:   "Create a pull request",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagTitle == "" {
				return fmt.Errorf("--title is required")
			}
			if flagHead == "" {
				return fmt.Errorf("--head is required")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.CreatePROpts{
				Title:     flagTitle,
				Body:      flagBody,
				Head:      flagHead,
				Base:      flagBase,
				Draft:     flagDraft,
				Reviewers: flagReviewers,
				Assignees: flagAssignees,
				Labels:    flagLabels,
				Milestone: flagMilestone,
			}

			pr, err := forge.PullRequests().Create(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return fmt.Errorf("creating pull request: %w", err)
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(pr)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s\n", pr.HTMLURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagTitle, "title", "t", "", "PR title")
	cmd.Flags().StringVarP(&flagBody, "body", "b", "", "PR body")
	cmd.Flags().StringVarP(&flagHead, "head", "H", "", "Head branch")
	cmd.Flags().StringVarP(&flagBase, "base", "B", "", "Base branch")
	cmd.Flags().BoolVarP(&flagDraft, "draft", "d", false, "Create as draft")
	cmd.Flags().StringSliceVarP(&flagReviewers, "reviewer", "r", nil, "Request a reviewer")
	cmd.Flags().StringSliceVarP(&flagAssignees, "assignee", "a", nil, "Assign to a user")
	cmd.Flags().StringSliceVarP(&flagLabels, "label", "l", nil, "Add a label")
	cmd.Flags().StringSliceVar(&flagLabels, "labels", nil, "Add a label")
	_ = cmd.Flags().MarkHidden("labels")
	cmd.Flags().StringVarP(&flagMilestone, "milestone", "m", "", "Assign to a milestone")
	return cmd
}

func prCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close <number>",
		Short: "Close a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if err := forge.PullRequests().Close(cmd.Context(), owner, repoName, number); err != nil {
				return fmt.Errorf("closing PR #%d: %w", number, err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Closed #%d\n", number)
			return nil
		},
	}
}

func prReopenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reopen <number>",
		Short: "Reopen a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if err := forge.PullRequests().Reopen(cmd.Context(), owner, repoName, number); err != nil {
				return fmt.Errorf("reopening PR #%d: %w", number, err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Reopened #%d\n", number)
			return nil
		},
	}
}

func prEditCmd() *cobra.Command {
	var (
		flagTitle     string
		flagBody      string
		flagBase      string
		flagReviewers []string
		flagAssignees []string
		flagLabels    []string
	)

	cmd := &cobra.Command{
		Use:   "edit <number>",
		Short: "Edit a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.UpdatePROpts{}
			if cmd.Flags().Changed("title") {
				opts.Title = &flagTitle
			}
			if cmd.Flags().Changed("body") {
				opts.Body = &flagBody
			}
			if cmd.Flags().Changed("base") {
				opts.Base = &flagBase
			}
			if cmd.Flags().Changed("reviewer") {
				opts.Reviewers = flagReviewers
			}
			if cmd.Flags().Changed("assignee") {
				opts.Assignees = flagAssignees
			}
			if cmd.Flags().Changed("label") || cmd.Flags().Changed("labels") {
				opts.Labels = flagLabels
			}

			pr, err := forge.PullRequests().Update(cmd.Context(), owner, repoName, number, opts)
			if err != nil {
				return fmt.Errorf("updating PR #%d: %w", number, err)
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(pr)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s\n", pr.HTMLURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagTitle, "title", "t", "", "Set the title")
	cmd.Flags().StringVarP(&flagBody, "body", "b", "", "Set the body")
	cmd.Flags().StringVarP(&flagBase, "base", "B", "", "Set the base branch")
	cmd.Flags().StringSliceVarP(&flagReviewers, "reviewer", "r", nil, "Set reviewers")
	cmd.Flags().StringSliceVarP(&flagAssignees, "assignee", "a", nil, "Set assignees")
	cmd.Flags().StringSliceVarP(&flagLabels, "label", "l", nil, "Set labels")
	cmd.Flags().StringSliceVar(&flagLabels, "labels", nil, "Set labels")
	_ = cmd.Flags().MarkHidden("labels")
	return cmd
}

func prMergeCmd() *cobra.Command {
	var (
		flagMethod  string
		flagTitle   string
		flagMessage string
		flagDelete  bool
	)

	cmd := &cobra.Command{
		Use:   "merge <number>",
		Short: "Merge a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.MergePROpts{
				Method:  flagMethod,
				Title:   flagTitle,
				Message: flagMessage,
				Delete:  flagDelete,
			}

			if err := forge.PullRequests().Merge(cmd.Context(), owner, repoName, number, opts); err != nil {
				return fmt.Errorf("merging PR #%d: %w", number, err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Merged #%d\n", number)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagMethod, "method", "m", "", "Merge method: merge, squash, rebase")
	cmd.Flags().StringVar(&flagTitle, "commit-title", "", "Commit title")
	cmd.Flags().StringVar(&flagMessage, "commit-message", "", "Commit message")
	cmd.Flags().BoolVarP(&flagDelete, "delete-branch", "d", false, "Delete branch after merge")
	return cmd
}

func prDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <number>",
		Short: "Show the diff of a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			diff, err := forge.PullRequests().Diff(cmd.Context(), owner, repoName, number)
			if err != nil {
				return fmt.Errorf("getting diff for PR #%d: %w", number, err)
			}

			_, _ = fmt.Fprint(os.Stdout, diff)
			return nil
		},
	}
}

func prCommentCmd() *cobra.Command {
	var flagBody string

	cmd := &cobra.Command{
		Use:   "comment <number>",
		Short: "Add a comment to a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}

			if flagBody == "" {
				return fmt.Errorf("--body is required")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			comment, err := forge.PullRequests().CreateComment(cmd.Context(), owner, repoName, number, flagBody)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(comment)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s\n", comment.HTMLURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagBody, "body", "b", "", "Comment body")
	return cmd
}

func prCheckoutCmd() *cobra.Command {
	var (
		flagRemoteName string
		flagBranch     string
		flagDetach     bool
		flagForce      bool
	)

	cmd := &cobra.Command{
		Use:   "checkout <number-or-url>",
		Short: "Check out a pull request locally",
		Long: `Check out a pull request's head branch locally.

If the PR is from a fork, the fork repository is added as a remote
(named after the fork owner by default), and the branch is fetched
and checked out with upstream tracking configured.

For same-repo PRs, the branch is fetched and checked out.

The argument can be a PR number or a full URL:
  forge pr checkout 123
  forge pr checkout https://github.com/owner/repo/pull/123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				forge           forges.Forge
				owner, repoName string
				domain          string
				number          int
				err             error
			)

			// Try parsing as a number first
			if n, parseErr := strconv.Atoi(args[0]); parseErr == nil {
				number = n
				forge, owner, repoName, domain, err = resolve.Repo(flagRepo, flagForgeType)
				if err != nil {
					return err
				}
			} else {
				// Try parsing as a URL
				var ref *forges.ResourceRef
				forge, domain, ref, err = resolve.ResourceFromURL(args[0])
				if err != nil {
					return fmt.Errorf("invalid PR number or URL %q: %w", args[0], err)
				}
				if ref.Type != forges.ResourceTypePR {
					return fmt.Errorf("URL does not point to a pull request: %s", args[0])
				}
				owner, repoName, number = ref.Owner, ref.Repo, ref.Number
			}

			ctx := cmd.Context()

			pr, err := forge.PullRequests().Get(ctx, owner, repoName, number)
			if err != nil {
				return fmt.Errorf("getting PR #%d: %w", number, err)
			}

			// remoteRef is usually the PR's head branch name, but Gitea/Forgejo
			// report a refs/pull/<n>/head ref when there is no head branch
			// (AGit-flow PRs, or PRs whose branch was deleted). Such a ref only
			// lives on the base repo.
			remoteRef := pr.Head.Ref

			// localBranch is what we'll name the local branch (defaults to the
			// head branch name, or pr-<number> when only a pull ref is known).
			localBranch := flagBranch
			if localBranch == "" {
				localBranch = defaultLocalBranch(pr)
			}

			// A pull ref isn't present on the fork remote, only on origin, so
			// route it through the same-repo path even for fork PRs.
			if pr.Head.Fork != nil && !isFullRef(remoteRef) {
				err = checkoutForkPR(ctx, domain, pr, remoteRef, localBranch, flagRemoteName, flagDetach, flagForce)
			} else {
				err = checkoutSameRepoPR(ctx, remoteRef, localBranch, flagDetach, flagForce)
			}
			if err != nil {
				return err
			}

			// Cache the PR number for the branch only after a successful
			// checkout, so a failed checkout doesn't leave a stale entry.
			if !flagDetach {
				_ = storePRForBranch(ctx, localBranch, number)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&flagRemoteName, "remote-name", "", "Name for fork remote (default: fork owner)")
	cmd.Flags().StringVarP(&flagBranch, "branch", "b", "", "Local branch name (default: same as remote)")
	cmd.Flags().BoolVar(&flagDetach, "detach", false, "Checkout in detached HEAD mode")
	cmd.Flags().BoolVarP(&flagForce, "force", "f", false, "Reset the local branch to the remote state even if it has diverged")
	return cmd
}

func checkoutForkPR(ctx context.Context, domain string, pr *forges.PullRequest, remoteRef, localBranch, flagRemoteName string, detach, force bool) error {
	fork := pr.Head.Fork
	remoteName := flagRemoteName
	if remoteName == "" {
		remoteName = fork.Owner
	}
	if remoteName == "" {
		remoteName = "fork"
	}

	url := cloneURL(domain, fork.CloneURL, fork.SSHURL)
	if url == "" {
		return fmt.Errorf("no clone URL available for fork repository")
	}

	remoteName, err := ensureRemote(ctx, remoteName, url)
	if err != nil {
		return err
	}

	return gitCheckout(ctx, remoteName, remoteRef, localBranch, detach, force)
}

func checkoutSameRepoPR(ctx context.Context, remoteRef, localBranch string, detach, force bool) error {
	return gitCheckout(ctx, resolve.RemoteName(), remoteRef, localBranch, detach, force)
}

func ensureRemote(ctx context.Context, preferredName, cloneURL string) (string, error) {
	remotes, err := exec.CommandContext(ctx, "git", "remote", "-v").Output()
	if err == nil {
		for _, line := range strings.Split(string(remotes), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && remoteMatches(fields[1], cloneURL) {
				return fields[0], nil
			}
		}
	}

	existingURL, err := exec.CommandContext(ctx, "git", "remote", "get-url", preferredName).Output()
	if err != nil {
		addCmd := exec.CommandContext(ctx, "git", "remote", "add", "--", preferredName, cloneURL)
		addCmd.Stdout = os.Stdout
		addCmd.Stderr = os.Stderr
		if err := addCmd.Run(); err != nil {
			return "", fmt.Errorf("adding remote %s: %w", preferredName, err)
		}
		return preferredName, nil
	}

	if remoteMatches(strings.TrimSpace(string(existingURL)), cloneURL) {
		return preferredName, nil
	}

	return "", fmt.Errorf("remote %q already exists with a different URL; use --remote-name to specify a different name", preferredName)
}

// remoteMatches reports whether existingURL points to the same repo as wantURL.
// It first checks for an exact match, then falls back to comparing domain/owner/repo
// so that SSH and HTTPS URLs for the same repository are treated as equivalent.
func remoteMatches(existingURL, wantURL string) bool {
	if existingURL == wantURL {
		return true
	}
	wantDomain, wantOwner, wantRepo, err := forges.ParseRepoURL(wantURL)
	if err != nil {
		return false
	}
	domain, owner, repo, err := forges.ParseRepoURL(existingURL)
	if err != nil {
		return false
	}
	return domain == wantDomain && owner == wantOwner && repo == wantRepo
}

// isFullRef reports whether ref is a fully-qualified git ref (e.g.
// refs/pull/<n>/head) rather than a bare branch name.
func isFullRef(ref string) bool {
	return strings.HasPrefix(ref, "refs/")
}

// defaultLocalBranch picks the local branch name for a checked-out PR. It uses
// the head branch name when available, but falls back to pr-<number> when only
// a pull ref is known (Gitea/Forgejo PRs with no head branch, e.g. AGit flow).
func defaultLocalBranch(pr *forges.PullRequest) string {
	if isFullRef(pr.Head.Ref) {
		return fmt.Sprintf("pr-%d", pr.Number)
	}
	return pr.Head.Ref
}

func gitCheckout(ctx context.Context, remote, remoteRef, localBranch string, detach, force bool) error {
	// A bare branch name needs the refs/heads/ prefix; a full ref (e.g.
	// refs/pull/<n>/head) is fetched as-is. The remote-tracking ref is named
	// after localBranch so a pull ref doesn't leak into refs/remotes/.
	src := remoteRef
	if !isFullRef(src) {
		src = "refs/heads/" + src
	}
	refspec := fmt.Sprintf("+%s:refs/remotes/%s/%s", src, remote, localBranch)
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "--", remote, refspec)
	fetchCmd.Stdout = os.Stdout
	fetchCmd.Stderr = os.Stderr
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("fetching %s/%s: %w", remote, remoteRef, err)
	}

	ref := remote + "/" + localBranch

	if detach {
		cmd := exec.CommandContext(ctx, "git", "checkout", "--detach", ref)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Try creating a new branch
	if exec.CommandContext(ctx, "git", "checkout", "-b", localBranch, ref).Run() == nil {
		return nil
	}

	// Branch exists - switch to it and try to fast-forward
	cmd := exec.CommandContext(ctx, "git", "checkout", localBranch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("checking out %s: %w", localBranch, err)
	}

	if exec.CommandContext(ctx, "git", "merge", "--ff-only", ref).Run() == nil {
		return nil
	}

	if !force {
		return fmt.Errorf("local branch %q has diverged from %s; use --force to reset it", localBranch, ref)
	}

	_, _ = fmt.Fprintf(os.Stderr, "warning: resetting %q to %s (local commits will be lost)\n", localBranch, ref)
	resetCmd := exec.CommandContext(ctx, "git", "reset", "--hard", ref)
	resetCmd.Stdout = os.Stdout
	resetCmd.Stderr = os.Stderr
	return resetCmd.Run()
}

func findPRForCurrentBranch(ctx context.Context, f forges.Forge, owner, repo string) (int, error) {
	out, err := exec.CommandContext(ctx, "git", "branch", "--show-current").Output()
	if err != nil {
		return 0, fmt.Errorf("getting current branch: %w (not in a git repository?)", err)
	}
	localBranch := strings.TrimSpace(string(out))
	if localBranch == "" {
		return 0, fmt.Errorf("not on a branch (detached HEAD state)")
	}

	// Check cache first (set by 'pr checkout')
	if n, err := loadPRForBranch(ctx, localBranch); err == nil {
		return n, nil
	}

	// If that yields nothing, fall back to API query. This API call is really
	// slow for Gitea since the Head filter is not actually implemented.
	//
	// This path assumes the local branch name matches the PR's remote head ref.
	// That breaks if the branch was checked out under a different name (e.g.
	// 'pr checkout 42 --branch myfork'), since the filter below compares against
	// localBranch. Such checkouts rely on the cache above being intact; there's
	// no way to recover the real head ref here once the cache is gone.
	headOwner := owner
	if remoteOwner, err := resolve.OwnerForBranch(ctx, localBranch); err == nil {
		headOwner = remoteOwner
	}

	// TODO: Limit 100 with no pagination means repos with >100 PRs may miss
	// the match on a fresh checkout (cache hides this in normal use).
	prs, err := f.PullRequests().List(ctx, owner, repo, forges.ListPROpts{
		Head:  headOwner + ":" + localBranch,
		State: "all",
		Limit: 100,
	})
	if err != nil {
		return 0, fmt.Errorf("listing PRs for branch %q: %w", localBranch, err)
	}

	// Need to filter the results again for owner:branch since the API results
	// don't respect the filter in case of Gitea.
	var matching []forges.PullRequest
	for _, pr := range prs {
		prHeadOwner := owner
		if pr.Head.Fork != nil {
			prHeadOwner = pr.Head.Fork.Owner
		}
		if pr.Head.Ref == localBranch && prHeadOwner == headOwner {
			matching = append(matching, pr)
		}
	}

	if len(matching) < 1 {
		return 0, fmt.Errorf("no pull request found for branch %q", localBranch)
	}

	// Prefer open PRs over closed/merged ones (a branch may be reused)
	for _, pr := range matching {
		if pr.State == "open" {
			// Store the PR number into local git config so that the next 'forge
			// pr view' call is a lot faster.
			_ = storePRForBranch(ctx, localBranch, pr.Number)
			return pr.Number, nil
		}
	}

	// No open PR, return the first match (closed/merged) but don't cache it
	return matching[0].Number, nil
}

func storePRForBranch(ctx context.Context, branch string, number int) error {
	key := fmt.Sprintf("branch.%s.forge-pr", branch)
	return exec.CommandContext(ctx, "git", "config", "--local", key, strconv.Itoa(number)).Run()
}

var prRefRE = regexp.MustCompile(`^refs/pull/(\d+)/head$`)

func loadPRForBranch(ctx context.Context, branch string) (int, error) {
	key := fmt.Sprintf("branch.%s.forge-pr", branch)
	out, err := exec.CommandContext(ctx, "git", "config", "--get", key).Output()
	if err == nil {
		return strconv.Atoi(strings.TrimSpace(string(out)))
	}

	// Fall back to gh CLI's format (refs/pull/<n>/head in branch.<name>.merge).
	// The regex only matches refs/pull/<n>/head, so refs/heads/* values are
	// safely rejected.
	mergeKey := fmt.Sprintf("branch.%s.merge", branch)
	out, err = exec.CommandContext(ctx, "git", "config", "--get", mergeKey).Output()
	if err != nil {
		return 0, err
	}
	m := prRefRE.FindStringSubmatch(strings.TrimSpace(string(out)))
	if m == nil {
		return 0, fmt.Errorf("not a PR ref")
	}
	return strconv.Atoi(m[1])
}

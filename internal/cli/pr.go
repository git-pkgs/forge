package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	forges "github.com/git-pkgs/forge"
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
	)

	cmd := &cobra.Command{
		Use:   "view <number>",
		Short: "View a pull request",
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
		Use:   "checkout <number>",
		Short: "Check out a pull request locally",
		Long: `Check out a pull request's head branch locally.

If the PR is from a fork, the fork repository is added as a remote
(named after the fork owner by default), and the branch is fetched
and checked out with upstream tracking configured.

For same-repo PRs, the branch is fetched and checked out.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}

			forge, owner, repoName, domain, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			pr, err := forge.PullRequests().Get(ctx, owner, repoName, number)
			if err != nil {
				return fmt.Errorf("getting PR #%d: %w", number, err)
			}

			// remoteRef is the branch name on the remote (PR's head branch)
			remoteRef := pr.Head.Ref

			// localBranch is what we'll name the local branch (defaults to remote ref)
			localBranch := remoteRef
			if flagBranch != "" {
				localBranch = flagBranch
			}

			if pr.Head.Fork != nil {
				return checkoutForkPR(ctx, domain, pr, remoteRef, localBranch, flagRemoteName, flagDetach, flagForce)
			}

			return checkoutSameRepoPR(ctx, remoteRef, localBranch, flagDetach, flagForce)
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

func gitCheckout(ctx context.Context, remote, remoteRef, localBranch string, detach, force bool) error {
	refspec := fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", remoteRef, remote, remoteRef)
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "--", remote, refspec)
	fetchCmd.Stdout = os.Stdout
	fetchCmd.Stderr = os.Stderr
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("fetching %s/%s: %w", remote, remoteRef, err)
	}

	ref := remote + "/" + remoteRef

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

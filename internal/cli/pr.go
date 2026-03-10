package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
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
}

func prViewCmd() *cobra.Command {
	var flagComments bool

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

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(pr)
			}

			_, _ = fmt.Fprintf(os.Stdout, "#%d %s\n", pr.Number, pr.Title)
			_, _ = fmt.Fprintf(os.Stdout, "State:   %s\n", pr.State)
			_, _ = fmt.Fprintf(os.Stdout, "Author:  %s\n", pr.Author.Login)
			_, _ = fmt.Fprintf(os.Stdout, "Branch:  %s -> %s\n", pr.Head, pr.Base)

			if pr.Draft {
				_, _ = fmt.Fprintln(os.Stdout, "Draft:   yes")
			}

			if len(pr.Reviewers) > 0 {
				names := make([]string, len(pr.Reviewers))
				for i, r := range pr.Reviewers {
					names[i] = r.Login
				}
				_, _ = fmt.Fprintf(os.Stdout, "Review:  %s\n", strings.Join(names, ", "))
			}

			if len(pr.Labels) > 0 {
				names := make([]string, len(pr.Labels))
				for i, l := range pr.Labels {
					names[i] = l.Name
				}
				_, _ = fmt.Fprintf(os.Stdout, "Labels:  %s\n", strings.Join(names, ", "))
			}

			if pr.Milestone != nil {
				_, _ = fmt.Fprintf(os.Stdout, "Mile:    %s\n", pr.Milestone.Title)
			}

			if pr.Additions > 0 || pr.Deletions > 0 {
				_, _ = fmt.Fprintf(os.Stdout, "Changes: +%d -%d (%d files)\n", pr.Additions, pr.Deletions, pr.ChangedFiles)
			}

			if pr.Body != "" {
				_, _ = fmt.Fprintln(os.Stdout)
				_, _ = fmt.Fprintln(os.Stdout, pr.Body)
			}

			if flagComments {
				comments, err := forge.PullRequests().ListComments(cmd.Context(), owner, repoName, number)
				if err != nil {
					return err
				}
				for _, c := range comments {
					_, _ = fmt.Fprintln(os.Stdout)
					_, _ = fmt.Fprintf(os.Stdout, "--- %s ---\n", c.Author.Login)
					_, _ = fmt.Fprintln(os.Stdout, c.Body)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagComments, "comments", "c", false, "Show comments")
	return cmd
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
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pull requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
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
					lines[i] = fmt.Sprintf("%d\t%s", pr.Number, pr.Title)
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"#", "TITLE", "AUTHOR", "HEAD", "UPDATED"}
			rows := make([][]string, len(prs))
			for i, pr := range prs {
				title := pr.Title
				if len(title) > 60 {
					title = title[:57] + "..."
				}
				rows[i] = []string{
					strconv.Itoa(pr.Number),
					title,
					pr.Author.Login,
					pr.Head,
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
	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 30, "Maximum number of PRs")
	cmd.Flags().StringVar(&flagSort, "sort", "", "Sort by: created, updated")
	cmd.Flags().StringVar(&flagOrder, "order", "", "Sort order: asc, desc")
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
		Use:   "create",
		Short: "Create a pull request",
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
			if cmd.Flags().Changed("label") {
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

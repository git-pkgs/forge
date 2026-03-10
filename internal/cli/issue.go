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

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Manage issues",
}

func init() {
	rootCmd.AddCommand(issueCmd)
	issueCmd.AddCommand(issueViewCmd())
	issueCmd.AddCommand(issueListCmd())
	issueCmd.AddCommand(issueCreateCmd())
	issueCmd.AddCommand(issueCloseCmd())
	issueCmd.AddCommand(issueReopenCmd())
	issueCmd.AddCommand(issueEditCmd())
	issueCmd.AddCommand(issueDeleteCmd())
	issueCmd.AddCommand(issueCommentCmd())
}

func issueViewCmd() *cobra.Command {
	var flagComments bool

	cmd := &cobra.Command{
		Use:   "view <number>",
		Short: "View an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			issue, err := forge.Issues().Get(cmd.Context(), owner, repoName, number)
			if err != nil {
				return fmt.Errorf("getting issue #%d: %w", number, err)
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(issue)
			}

			_, _ = fmt.Fprintf(os.Stdout, "#%d %s\n", issue.Number, issue.Title)
			_, _ = fmt.Fprintf(os.Stdout, "State:   %s\n", issue.State)
			_, _ = fmt.Fprintf(os.Stdout, "Author:  %s\n", issue.Author.Login)

			if len(issue.Assignees) > 0 {
				names := make([]string, len(issue.Assignees))
				for i, a := range issue.Assignees {
					names[i] = a.Login
				}
				_, _ = fmt.Fprintf(os.Stdout, "Assign:  %s\n", strings.Join(names, ", "))
			}

			if len(issue.Labels) > 0 {
				names := make([]string, len(issue.Labels))
				for i, l := range issue.Labels {
					names[i] = l.Name
				}
				_, _ = fmt.Fprintf(os.Stdout, "Labels:  %s\n", strings.Join(names, ", "))
			}

			if issue.Milestone != nil {
				_, _ = fmt.Fprintf(os.Stdout, "Mile:    %s\n", issue.Milestone.Title)
			}

			if issue.Body != "" {
				_, _ = fmt.Fprintln(os.Stdout)
				_, _ = fmt.Fprintln(os.Stdout, issue.Body)
			}

			if flagComments {
				comments, err := forge.Issues().ListComments(cmd.Context(), owner, repoName, number)
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

func issueListCmd() *cobra.Command {
	var (
		flagState    string
		flagAssignee string
		flagAuthor   string
		flagLabels   []string
		flagLimit    int
		flagSort     string
		flagOrder    string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.ListIssueOpts{
				State:    flagState,
				Assignee: flagAssignee,
				Author:   flagAuthor,
				Labels:   flagLabels,
				Sort:     flagSort,
				Order:    flagOrder,
				Limit:    flagLimit,
			}

			issues, err := forge.Issues().List(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return fmt.Errorf("listing issues: %w", err)
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(issues)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(issues))
				for i, iss := range issues {
					lines[i] = fmt.Sprintf("%d\t%s", iss.Number, iss.Title)
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"#", "TITLE", "AUTHOR", "LABELS", "UPDATED"}
			rows := make([][]string, len(issues))
			for i, iss := range issues {
				labels := make([]string, len(iss.Labels))
				for j, l := range iss.Labels {
					labels[j] = l.Name
				}
				title := iss.Title
				if len(title) > 60 {
					title = title[:57] + "..."
				}
				rows[i] = []string{
					strconv.Itoa(iss.Number),
					title,
					iss.Author.Login,
					strings.Join(labels, ", "),
					iss.UpdatedAt.Format("2006-01-02"),
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagState, "state", "s", "open", "Filter by state: open, closed, all")
	cmd.Flags().StringVarP(&flagAssignee, "assignee", "a", "", "Filter by assignee")
	cmd.Flags().StringVarP(&flagAuthor, "author", "A", "", "Filter by author")
	cmd.Flags().StringSliceVarP(&flagLabels, "label", "l", nil, "Filter by label")
	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 30, "Maximum number of issues")
	cmd.Flags().StringVar(&flagSort, "sort", "", "Sort by: created, updated, comments")
	cmd.Flags().StringVar(&flagOrder, "order", "", "Sort order: asc, desc")
	return cmd
}

func issueCreateCmd() *cobra.Command {
	var (
		flagTitle     string
		flagBody      string
		flagAssignees []string
		flagLabels    []string
		flagMilestone string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagTitle == "" {
				return fmt.Errorf("--title is required")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.CreateIssueOpts{
				Title:     flagTitle,
				Body:      flagBody,
				Assignees: flagAssignees,
				Labels:    flagLabels,
				Milestone: flagMilestone,
			}

			issue, err := forge.Issues().Create(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return fmt.Errorf("creating issue: %w", err)
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(issue)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s\n", issue.HTMLURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagTitle, "title", "t", "", "Issue title")
	cmd.Flags().StringVarP(&flagBody, "body", "b", "", "Issue body")
	cmd.Flags().StringSliceVarP(&flagAssignees, "assignee", "a", nil, "Assign to a user")
	cmd.Flags().StringSliceVarP(&flagLabels, "label", "l", nil, "Add a label")
	cmd.Flags().StringVarP(&flagMilestone, "milestone", "m", "", "Assign to a milestone")
	return cmd
}

func issueCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close <number>",
		Short: "Close an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if err := forge.Issues().Close(cmd.Context(), owner, repoName, number); err != nil {
				return fmt.Errorf("closing issue #%d: %w", number, err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Closed #%d\n", number)
			return nil
		},
	}
}

func issueReopenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reopen <number>",
		Short: "Reopen an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if err := forge.Issues().Reopen(cmd.Context(), owner, repoName, number); err != nil {
				return fmt.Errorf("reopening issue #%d: %w", number, err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Reopened #%d\n", number)
			return nil
		},
	}
}

func issueEditCmd() *cobra.Command {
	var (
		flagTitle     string
		flagBody      string
		flagAssignees []string
		flagLabels    []string
		flagMilestone string
	)

	cmd := &cobra.Command{
		Use:   "edit <number>",
		Short: "Edit an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.UpdateIssueOpts{}
			if cmd.Flags().Changed("title") {
				opts.Title = &flagTitle
			}
			if cmd.Flags().Changed("body") {
				opts.Body = &flagBody
			}
			if cmd.Flags().Changed("assignee") {
				opts.Assignees = flagAssignees
			}
			if cmd.Flags().Changed("label") {
				opts.Labels = flagLabels
			}
			if cmd.Flags().Changed("milestone") {
				opts.Milestone = &flagMilestone
			}

			issue, err := forge.Issues().Update(cmd.Context(), owner, repoName, number, opts)
			if err != nil {
				return fmt.Errorf("updating issue #%d: %w", number, err)
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(issue)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s\n", issue.HTMLURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagTitle, "title", "t", "", "Set the title")
	cmd.Flags().StringVarP(&flagBody, "body", "b", "", "Set the body")
	cmd.Flags().StringSliceVarP(&flagAssignees, "assignee", "a", nil, "Set assignees")
	cmd.Flags().StringSliceVarP(&flagLabels, "label", "l", nil, "Set labels")
	cmd.Flags().StringVarP(&flagMilestone, "milestone", "m", "", "Set the milestone")
	return cmd
}

func issueDeleteCmd() *cobra.Command {
	var flagYes bool

	cmd := &cobra.Command{
		Use:   "delete <number>",
		Short: "Delete an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if !flagYes {
				if err := confirm(fmt.Sprintf("Delete issue #%d in %s/%s? [y/N] ", number, owner, repoName)); err != nil {
					return err
				}
			}

			if err := forge.Issues().Delete(cmd.Context(), owner, repoName, number); err != nil {
				return fmt.Errorf("deleting issue #%d: %w", number, err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Deleted #%d\n", number)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func issueCommentCmd() *cobra.Command {
	var flagBody string

	cmd := &cobra.Command{
		Use:   "comment <number>",
		Short: "Add a comment to an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}

			if flagBody == "" {
				return fmt.Errorf("--body is required")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			comment, err := forge.Issues().CreateComment(cmd.Context(), owner, repoName, number, flagBody)
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

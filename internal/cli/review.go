package cli

import (
	"fmt"
	"os"
	"strconv"

	forges "github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Manage pull request reviews",
}

var reviewerCmd = &cobra.Command{
	Use:   "reviewer",
	Short: "Manage pull request reviewers",
}

func init() {
	prCmd.AddCommand(reviewCmd)
	prCmd.AddCommand(reviewerCmd)
	reviewCmd.AddCommand(reviewListCmd())
	reviewCmd.AddCommand(reviewApproveCmd())
	reviewCmd.AddCommand(reviewRejectCmd())
	reviewerCmd.AddCommand(reviewerRequestCmd())
	reviewerCmd.AddCommand(reviewerRemoveCmd())
}

func reviewListCmd() *cobra.Command {
	var flagLimit int

	cmd := &cobra.Command{
		Use:   "list <number>",
		Short: "List reviews on a pull request",
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

			reviews, err := forge.Reviews().List(cmd.Context(), owner, repoName, number, forges.ListReviewOpts{
				Limit: flagLimit,
			})
			if err != nil {
				return notSupported(err, "PR reviews")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(reviews)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(reviews))
				for i, r := range reviews {
					lines[i] = fmt.Sprintf("%s\t%s", r.Author.Login, r.State)
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"AUTHOR", "STATE", "BODY"}
			rows := make([][]string, len(reviews))
			for i, r := range reviews {
				body := r.Body
				if len(body) > 60 {
					body = body[:57] + "..."
				}
				rows[i] = []string{
					r.Author.Login,
					string(r.State),
					body,
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 30, "Maximum number of reviews")
	return cmd
}

func reviewApproveCmd() *cobra.Command {
	var flagBody string

	cmd := &cobra.Command{
		Use:   "approve <number>",
		Short: "Approve a pull request",
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

			review, err := forge.Reviews().Submit(cmd.Context(), owner, repoName, number, forges.SubmitReviewOpts{
				State: forges.ReviewApproved,
				Body:  flagBody,
			})
			if err != nil {
				return notSupported(err, "PR approval")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(review)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Approved #%d\n", number)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagBody, "body", "b", "", "Review body")
	return cmd
}

func reviewRejectCmd() *cobra.Command {
	var flagBody string

	cmd := &cobra.Command{
		Use:   "reject <number>",
		Short: "Request changes on a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}

			if flagBody == "" {
				return fmt.Errorf("--body is required when requesting changes")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			review, err := forge.Reviews().Submit(cmd.Context(), owner, repoName, number, forges.SubmitReviewOpts{
				State: forges.ReviewChangesRequested,
				Body:  flagBody,
			})
			if err != nil {
				return notSupported(err, "requesting changes")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(review)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Requested changes on #%d\n", number)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagBody, "body", "b", "", "Review body")
	return cmd
}

func reviewerRequestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "request <number> <users...>",
		Short: "Request reviewers on a pull request",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if err := forge.Reviews().RequestReviewers(cmd.Context(), owner, repoName, number, args[1:]); err != nil {
				return notSupported(err, "requesting reviewers")
			}

			_, _ = fmt.Fprintf(os.Stdout, "Requested reviewers on #%d\n", number)
			return nil
		},
	}
}

func reviewerRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <number> <users...>",
		Short: "Remove reviewer requests from a pull request",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if err := forge.Reviews().RemoveReviewers(cmd.Context(), owner, repoName, number, args[1:]); err != nil {
				return notSupported(err, "removing reviewers")
			}

			_, _ = fmt.Fprintf(os.Stdout, "Removed reviewers from #%d\n", number)
			return nil
		},
	}
}

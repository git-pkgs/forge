package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

var commentCmd = &cobra.Command{
	Use:   "comment <number>",
	Short: "Add a comment to an issue or pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		number, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid number: %s", args[0])
		}

		if flagCommentBody == "" {
			return fmt.Errorf("--body is required")
		}

		forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
		if err != nil {
			return err
		}

		// Try issue first, fall back to PR
		comment, issueErr := forge.Issues().CreateComment(cmd.Context(), owner, repoName, number, flagCommentBody)
		if issueErr != nil {
			var prErr error
			comment, prErr = forge.PullRequests().CreateComment(cmd.Context(), owner, repoName, number, flagCommentBody)
			if prErr != nil {
				return fmt.Errorf("could not comment on issue (%v) or pull request (%v)", issueErr, prErr)
			}
		}

		p := printer()
		if p.Format == output.JSON {
			return p.PrintJSON(comment)
		}

		fmt.Fprintf(os.Stdout, "%s\n", comment.HTMLURL)
		return nil
	},
}

var flagCommentBody string

func init() {
	rootCmd.AddCommand(commentCmd)
	commentCmd.Flags().StringVarP(&flagCommentBody, "body", "b", "", "Comment body")
}

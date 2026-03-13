package cli

import (
	"fmt"

	forges "github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Manage commit statuses",
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.AddCommand(statusListCmd())
	statusCmd.AddCommand(statusSetCmd())
}

func statusListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <sha>",
		Short: "List commit statuses",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sha := args[0]

			f, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			statuses, err := f.CommitStatuses().List(cmd.Context(), owner, repoName, sha)
			if err != nil {
				return notSupported(err, "list commit statuses")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(statuses)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(statuses))
				for i, s := range statuses {
					lines[i] = fmt.Sprintf("%s\t%s", s.State, s.Context)
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"STATE", "CONTEXT", "DESCRIPTION", "URL"}
			rows := make([][]string, len(statuses))
			for i, s := range statuses {
				rows[i] = []string{
					s.State,
					s.Context,
					s.Description,
					s.TargetURL,
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}
	return cmd
}

func statusSetCmd() *cobra.Command {
	var (
		flagState       string
		flagContext     string
		flagDescription string
		flagURL         string
	)

	cmd := &cobra.Command{
		Use:   "set <sha>",
		Short: "Set a commit status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sha := args[0]

			f, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.SetCommitStatusOpts{
				State:       flagState,
				Context:     flagContext,
				Description: flagDescription,
				TargetURL:   flagURL,
			}

			status, err := f.CommitStatuses().Set(cmd.Context(), owner, repoName, sha, opts)
			if err != nil {
				return notSupported(err, "set commit status")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(status)
			}

			fmt.Printf("%s %s\n", status.State, status.Context)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagState, "state", "", "Status state: success, failure, pending, error (required)")
	cmd.Flags().StringVar(&flagContext, "context", "", "Status context, e.g. \"my-check\" (required)")
	cmd.Flags().StringVar(&flagDescription, "description", "", "Short description")
	cmd.Flags().StringVar(&flagURL, "url", "", "Target URL for details")
	_ = cmd.MarkFlagRequired("state")
	_ = cmd.MarkFlagRequired("context")
	return cmd
}

package cli

import (
	"fmt"
	"os"

	"github.com/git-pkgs/forges/internal/output"
	"github.com/git-pkgs/forges/internal/resolve"
	"github.com/git-pkgs/forges"
	"github.com/spf13/cobra"
)

var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "Manage branches",
}

func init() {
	rootCmd.AddCommand(branchCmd)
	branchCmd.AddCommand(branchListCmd())
	branchCmd.AddCommand(branchCreateCmd())
	branchCmd.AddCommand(branchDeleteCmd())
}

func branchListCmd() *cobra.Command {
	var flagLimit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List branches",
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.ListBranchOpts{
				Limit: flagLimit,
			}

			branches, err := forge.Branches().List(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(branches)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(branches))
				for i, b := range branches {
					lines[i] = b.Name
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"NAME", "SHA", "PROTECTED"}
			rows := make([][]string, len(branches))
			for i, b := range branches {
				sha := b.SHA
				if len(sha) > 7 {
					sha = sha[:7]
				}
				rows[i] = []string{
					b.Name,
					sha,
					fmt.Sprintf("%v", b.Protected),
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 0, "Maximum number of branches")
	return cmd
}

func branchCreateCmd() *cobra.Command {
	var flagFrom string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			from := flagFrom
			if from == "" {
				from = "main"
			}

			branch, err := forge.Branches().Create(cmd.Context(), owner, repoName, name, from)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(branch)
			}

			fmt.Fprintf(os.Stdout, "%s %s\n", branch.Name, branch.SHA)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagFrom, "from", "", "Source branch or commit (default: main)")
	return cmd
}

func branchDeleteCmd() *cobra.Command {
	var flagYes bool

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if !flagYes {
				if err := confirm(fmt.Sprintf("Delete branch %q in %s/%s? [y/N] ", name, owner, repoName)); err != nil {
					return err
				}
			}

			if err := forge.Branches().Delete(cmd.Context(), owner, repoName, name); err != nil {
				return err
			}

			fmt.Fprintf(os.Stdout, "Deleted %s\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation")
	return cmd
}

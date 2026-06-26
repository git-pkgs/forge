package cli

import (
	"fmt"
	"os"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/git"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

const shortSHALength = 7

var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "Manage branches",
}

func init() {
	rootCmd.AddCommand(branchCmd)
	branchCmd.AddCommand(branchListCmd())
	branchCmd.AddCommand(branchCreateCmd())
	branchCmd.AddCommand(branchDeleteCmd())
	branchCmd.AddCommand(branchShowBaseCmd())
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
				return notSupported(err, "branches")
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
				if len(sha) > shortSHALength {
					sha = sha[:shortSHALength]
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
				return notSupported(err, "branches")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(branch)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s %s\n", branch.Name, branch.SHA)
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
				return notSupported(err, "branches")
			}

			_, _ = fmt.Fprintf(os.Stdout, "Deleted %s\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func branchShowBaseCmd() *cobra.Command {
	var flagRefresh bool

	cmd := &cobra.Command{
		Use:   "show-base [branch]",
		Short: "Show the base branch for a branch",
		Long: `Show the base branch for the specified branch (defaults to the current branch).

It first checks for a cached base branch name under the local git config key
'branch.<branch>.forge-merge-base' in .git/config. If not found, it queries the
forge API for an open pull request, caches the resolved target branch name back in
the local git configuration, and returns it.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var branch string
			if len(args) > 0 {
				branch = args[0]
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			base, err := git.GetOrFetchBaseBranch(cmd.Context(), forge, "", owner, repoName, branch, flagRefresh)
			if err != nil {
				return err
			}

			fmt.Println(base)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagRefresh, "refresh", false, "Force query the forge API and update cached base branch")
	return cmd
}

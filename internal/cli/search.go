package cli

import (
	"fmt"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

const maxSearchDescLength = 50

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search across the forge",
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.AddCommand(searchReposCmd())
}

func searchReposCmd() *cobra.Command {
	var (
		flagLimit int
		flagSort  string
		flagOrder string
	)

	cmd := &cobra.Command{
		Use:   "repos <query>",
		Short: "Search repositories",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, _, _, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.SearchRepoOpts{
				Query: args[0],
				Sort:  flagSort,
				Order: flagOrder,
				Limit: flagLimit,
			}

			repos, err := forge.Repos().Search(cmd.Context(), opts)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(repos)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(repos))
				for i, r := range repos {
					lines[i] = r.FullName
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"NAME", "DESCRIPTION", "STARS", "LANGUAGE"}
			rows := make([][]string, len(repos))
			for i, r := range repos {
				desc := r.Description
				if len(desc) > maxSearchDescLength {
					desc = desc[:47] + "..."
				}
				rows[i] = []string{
					r.FullName,
					desc,
					fmt.Sprintf("%d", r.StargazersCount),
					r.Language,
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagLimit, "limit", "L", defaultSearchLimit, "Maximum number of results")
	cmd.Flags().StringVar(&flagSort, "sort", "", "Sort field: stars, forks, updated")
	cmd.Flags().StringVar(&flagOrder, "order", "", "Sort order: asc, desc")
	return cmd
}

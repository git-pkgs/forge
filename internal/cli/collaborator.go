package cli

import (
	"fmt"
	"os"

	forges "github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

var collaboratorCmd = &cobra.Command{
	Use:     "collaborator",
	Aliases: []string{"collab"},
	Short:   "Manage collaborators",
}

func init() {
	rootCmd.AddCommand(collaboratorCmd)
	collaboratorCmd.AddCommand(collaboratorListCmd())
	collaboratorCmd.AddCommand(collaboratorAddCmd())
	collaboratorCmd.AddCommand(collaboratorRemoveCmd())
}

func collaboratorListCmd() *cobra.Command {
	var flagLimit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List collaborators",
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.ListCollaboratorOpts{
				Limit: flagLimit,
			}

			collabs, err := forge.Collaborators().List(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return notSupported(err, "collaborators")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(collabs)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(collabs))
				for i, c := range collabs {
					lines[i] = c.Login
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"LOGIN", "PERMISSION"}
			rows := make([][]string, len(collabs))
			for i, c := range collabs {
				rows[i] = []string{c.Login, c.Permission}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 0, "Maximum number of collaborators")
	return cmd
}

func collaboratorAddCmd() *cobra.Command {
	var flagPermission string

	cmd := &cobra.Command{
		Use:   "add <username>",
		Short: "Add a collaborator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.AddCollaboratorOpts{
				Permission: flagPermission,
			}

			if err := forge.Collaborators().Add(cmd.Context(), owner, repoName, username, opts); err != nil {
				return notSupported(err, "collaborators")
			}

			_, _ = fmt.Fprintf(os.Stdout, "Added %s\n", username)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagPermission, "permission", "p", "", "Permission level (e.g. push, pull, admin)")
	return cmd
}

func collaboratorRemoveCmd() *cobra.Command {
	var flagYes bool

	cmd := &cobra.Command{
		Use:   "remove <username>",
		Short: "Remove a collaborator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if !flagYes {
				if err := confirm(fmt.Sprintf("Remove collaborator %q from %s/%s? [y/N] ", username, owner, repoName)); err != nil {
					return err
				}
			}

			if err := forge.Collaborators().Remove(cmd.Context(), owner, repoName, username); err != nil {
				return notSupported(err, "collaborators")
			}

			_, _ = fmt.Fprintf(os.Stdout, "Removed %s\n", username)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation")
	return cmd
}

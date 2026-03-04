package cli

import (
	"fmt"
	"os"

	"github.com/git-pkgs/forges/internal/output"
	"github.com/git-pkgs/forges/internal/resolve"
	"github.com/git-pkgs/forges"
	"github.com/spf13/cobra"
)

var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage repository secrets",
}

func init() {
	rootCmd.AddCommand(secretCmd)
	secretCmd.AddCommand(secretListCmd())
	secretCmd.AddCommand(secretSetCmd())
	secretCmd.AddCommand(secretDeleteCmd())
}

func secretListCmd() *cobra.Command {
	var flagLimit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.ListSecretOpts{
				Limit: flagLimit,
			}

			secrets, err := forge.Secrets().List(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(secrets)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(secrets))
				for i, s := range secrets {
					lines[i] = s.Name
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"NAME", "UPDATED"}
			rows := make([][]string, len(secrets))
			for i, s := range secrets {
				updated := ""
				if !s.UpdatedAt.IsZero() {
					updated = s.UpdatedAt.Format("2006-01-02")
				}
				rows[i] = []string{s.Name, updated}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 0, "Maximum number of secrets")
	return cmd
}

func secretSetCmd() *cobra.Command {
	var (
		flagName  string
		flagValue string
	)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Create or update a secret",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagName == "" {
				return fmt.Errorf("--name is required")
			}
			if flagValue == "" {
				return fmt.Errorf("--value is required")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if err := forge.Secrets().Set(cmd.Context(), owner, repoName, forges.SetSecretOpts{
				Name:  flagName,
				Value: flagValue,
			}); err != nil {
				return err
			}

			fmt.Fprintf(os.Stdout, "Set %s\n", flagName)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagName, "name", "n", "", "Secret name")
	cmd.Flags().StringVarP(&flagValue, "value", "v", "", "Secret value")
	return cmd
}

func secretDeleteCmd() *cobra.Command {
	var flagYes bool

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if !flagYes {
				if err := confirm(fmt.Sprintf("Delete secret %q in %s/%s? [y/N] ", name, owner, repoName)); err != nil {
					return err
				}
			}

			if err := forge.Secrets().Delete(cmd.Context(), owner, repoName, name); err != nil {
				return err
			}

			fmt.Fprintf(os.Stdout, "Deleted %s\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation")
	return cmd
}

package cli

import (
	"fmt"
	"os"

	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/git-pkgs/forge"
	"github.com/spf13/cobra"
)

var deployKeyCmd = &cobra.Command{
	Use:   "deploy-key",
	Short: "Manage deploy keys",
}

func init() {
	rootCmd.AddCommand(deployKeyCmd)
	deployKeyCmd.AddCommand(deployKeyListCmd())
	deployKeyCmd.AddCommand(deployKeyAddCmd())
	deployKeyCmd.AddCommand(deployKeyDeleteCmd())
}

func deployKeyListCmd() *cobra.Command {
	var flagLimit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List deploy keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.ListDeployKeyOpts{
				Limit: flagLimit,
			}

			keys, err := forge.DeployKeys().List(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(keys)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(keys))
				for i, k := range keys {
					lines[i] = fmt.Sprintf("%d\t%s", k.ID, k.Title)
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"ID", "TITLE", "READ-ONLY"}
			rows := make([][]string, len(keys))
			for i, k := range keys {
				rows[i] = []string{
					fmt.Sprintf("%d", k.ID),
					k.Title,
					fmt.Sprintf("%v", k.ReadOnly),
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 0, "Maximum number of keys")
	return cmd
}

func deployKeyAddCmd() *cobra.Command {
	var (
		flagTitle    string
		flagKey      string
		flagReadOnly bool
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a deploy key",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagTitle == "" {
				return fmt.Errorf("--title is required")
			}
			if flagKey == "" {
				return fmt.Errorf("--key is required")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			key, err := forge.DeployKeys().Create(cmd.Context(), owner, repoName, forges.CreateDeployKeyOpts{
				Title:    flagTitle,
				Key:      flagKey,
				ReadOnly: flagReadOnly,
			})
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(key)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%d %s\n", key.ID, key.Title)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagTitle, "title", "t", "", "Key title")
	cmd.Flags().StringVarP(&flagKey, "key", "k", "", "Public key content")
	cmd.Flags().BoolVar(&flagReadOnly, "read-only", true, "Read-only access (default true)")
	return cmd
}

func deployKeyDeleteCmd() *cobra.Command {
	var flagYes bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a deploy key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var id int64
			if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
				return fmt.Errorf("invalid key ID: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if !flagYes {
				if err := confirm(fmt.Sprintf("Delete deploy key %d in %s/%s? [y/N] ", id, owner, repoName)); err != nil {
					return err
				}
			}

			if err := forge.DeployKeys().Delete(cmd.Context(), owner, repoName, id); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(os.Stdout, "Deleted key %d\n", id)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation")
	return cmd
}

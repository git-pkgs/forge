package cli

import (
	"fmt"
	"os"

	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/git-pkgs/forge"
	"github.com/spf13/cobra"
)

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "Manage labels",
}

func init() {
	rootCmd.AddCommand(labelCmd)
	labelCmd.AddCommand(labelListCmd())
	labelCmd.AddCommand(labelCreateCmd())
	labelCmd.AddCommand(labelEditCmd())
	labelCmd.AddCommand(labelDeleteCmd())
}

func labelListCmd() *cobra.Command {
	var flagLimit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List labels",
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.ListLabelOpts{
				Limit: flagLimit,
			}

			labels, err := forge.Labels().List(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(labels)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(labels))
				for i, l := range labels {
					lines[i] = l.Name
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"NAME", "COLOR", "DESCRIPTION"}
			rows := make([][]string, len(labels))
			for i, l := range labels {
				desc := l.Description
				if len(desc) > 50 {
					desc = desc[:47] + "..."
				}
				rows[i] = []string{l.Name, l.Color, desc}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 0, "Maximum number of labels")
	return cmd
}

func labelCreateCmd() *cobra.Command {
	var (
		flagName        string
		flagColor       string
		flagDescription string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a label",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagName == "" {
				return fmt.Errorf("--name is required")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.CreateLabelOpts{
				Name:        flagName,
				Color:       flagColor,
				Description: flagDescription,
			}

			label, err := forge.Labels().Create(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(label)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s\n", label.Name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagName, "name", "n", "", "Label name")
	cmd.Flags().StringVarP(&flagColor, "color", "c", "", "Label color (hex without #)")
	cmd.Flags().StringVarP(&flagDescription, "description", "d", "", "Label description")
	return cmd
}

func labelEditCmd() *cobra.Command {
	var (
		flagName        string
		flagColor       string
		flagDescription string
	)

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a label",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			labelName := args[0]

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.UpdateLabelOpts{}
			if cmd.Flags().Changed("name") {
				opts.Name = &flagName
			}
			if cmd.Flags().Changed("color") {
				opts.Color = &flagColor
			}
			if cmd.Flags().Changed("description") {
				opts.Description = &flagDescription
			}

			label, err := forge.Labels().Update(cmd.Context(), owner, repoName, labelName, opts)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(label)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s\n", label.Name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagName, "name", "n", "", "New label name")
	cmd.Flags().StringVarP(&flagColor, "color", "c", "", "New label color (hex without #)")
	cmd.Flags().StringVarP(&flagDescription, "description", "d", "", "New label description")
	return cmd
}

func labelDeleteCmd() *cobra.Command {
	var flagYes bool

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a label",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			labelName := args[0]

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if !flagYes {
				if err := confirm(fmt.Sprintf("Delete label %q in %s/%s? [y/N] ", labelName, owner, repoName)); err != nil {
					return err
				}
			}

			if err := forge.Labels().Delete(cmd.Context(), owner, repoName, labelName); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(os.Stdout, "Deleted %s\n", labelName)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation")
	return cmd
}

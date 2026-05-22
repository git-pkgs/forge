package cli

import (
	"errors"
	"fmt"
	"os"

	forges "github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

const maxLabelDescLength = 50

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
	var (
		flagLimit int
		flagWeb   bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List labels",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if flagWeb {
				repo, err := f.Repos().Get(cmd.Context(), owner, repoName)
				if err != nil {
					return fmt.Errorf("getting repository: %w", err)
				}
				return openBrowser(f.Labels().ListURL(repo.HTMLURL))
			}

			opts := forges.ListLabelOpts{
				Limit: flagLimit,
			}

			labels, err := f.Labels().List(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return notSupported(err, "labels")
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
				if len(desc) > maxLabelDescLength {
					desc = desc[:47] + "..."
				}
				rows[i] = []string{l.Name, l.Color, desc}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 0, "Maximum number of labels")
	cmd.Flags().BoolVarP(&flagWeb, "web", "w", false, "Open in browser")
	return cmd
}

func labelCreateCmd() *cobra.Command {
	var (
		flagName        string
		flagColor       string
		flagDescription string
		flagForce       bool
	)

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a label",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := flagName
			if len(args) > 0 {
				name = args[0]
			}
			if name == "" {
				return fmt.Errorf("label name is required (provide as argument or --name)")
			}

			f, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.CreateLabelOpts{
				Name:        name,
				Color:       flagColor,
				Description: flagDescription,
			}

			label, err := f.Labels().Create(cmd.Context(), owner, repoName, opts)
			if err != nil {
				if flagForce && errors.Is(err, forges.ErrLabelExists) {
					updateOpts := forges.UpdateLabelOpts{}
					if flagColor != "" {
						updateOpts.Color = &flagColor
					}
					if flagDescription != "" {
						updateOpts.Description = &flagDescription
					}
					label, err = f.Labels().Update(cmd.Context(), owner, repoName, name, updateOpts)
					if err != nil {
						return notSupported(err, "labels")
					}
				} else {
					return notSupported(err, "labels")
				}
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(label)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s\n", label.Name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagName, "name", "n", "", "Label name (can also be provided as first argument)")
	cmd.Flags().StringVarP(&flagColor, "color", "c", "", "Label color (hex without #)")
	cmd.Flags().StringVarP(&flagDescription, "description", "d", "", "Label description")
	cmd.Flags().BoolVarP(&flagForce, "force", "f", false, "Update the label if it already exists")
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
				return notSupported(err, "labels")
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
				return notSupported(err, "labels")
			}

			_, _ = fmt.Fprintf(os.Stdout, "Deleted %s\n", labelName)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation")
	return cmd
}

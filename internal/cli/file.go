package cli

import (
	"fmt"
	"os"

	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Read file content",
}

func init() {
	rootCmd.AddCommand(fileCmd)
	fileCmd.AddCommand(fileCatCmd())
	fileCmd.AddCommand(fileLsCmd())
}

func fileCatCmd() *cobra.Command {
	var flagRef string

	cmd := &cobra.Command{
		Use:   "cat <path>",
		Short: "Print file content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			file, err := forge.Files().Get(cmd.Context(), owner, repoName, path, flagRef)
			if err != nil {
				return notSupported(err, "file content")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(file)
			}

			_, _ = os.Stdout.Write(file.Content)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagRef, "ref", "", "Git ref (branch, tag, or SHA)")
	return cmd
}

func fileLsCmd() *cobra.Command {
	var flagRef string

	cmd := &cobra.Command{
		Use:   "ls [path]",
		Short: "List directory entries",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ""
			if len(args) > 0 {
				path = args[0]
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			entries, err := forge.Files().List(cmd.Context(), owner, repoName, path, flagRef)
			if err != nil {
				return notSupported(err, "file listing")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(entries)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(entries))
				for i, e := range entries {
					lines[i] = e.Name
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"TYPE", "NAME", "SIZE"}
			rows := make([][]string, len(entries))
			for i, e := range entries {
				size := ""
				if e.Type == "file" {
					size = formatSize(e.Size)
				}
				rows[i] = []string{e.Type, e.Name, size}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagRef, "ref", "", "Git ref (branch, tag, or SHA)")
	return cmd
}

func formatSize(b int64) string {
	if b == 0 {
		return "0"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

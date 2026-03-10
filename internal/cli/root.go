package cli

import (
	"errors"
	"fmt"
	"os"

	forges "github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/config"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/spf13/cobra"
)

var (
	flagRepo      string
	flagForgeType string
	flagOutput    string
)

var rootCmd = &cobra.Command{
	Use:   "forge",
	Short: "Work with git forges from the command line",
	Long:  "Supports GitHub, GitLab, Gitea, and Forgejo through a single interface.",
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !cmd.Flags().Changed("output") {
			cfg, err := config.Load()
			if err == nil && cfg != nil && cfg.Default.Output != "" {
				flagOutput = cfg.Default.Output
			}
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagRepo, "repo", "R", "", "Select a repository (OWNER/REPO)")
	rootCmd.PersistentFlags().StringVar(&flagForgeType, "forge-type", "", "Force forge type: github, gitlab, gitea, forgejo")
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "table", "Output format: table, json, plain")
}

// notSupported wraps ErrNotSupported with a user-friendly message
// describing which feature isn't available.
func notSupported(err error, feature string) error {
	if errors.Is(err, forges.ErrNotSupported) {
		return fmt.Errorf("%s is not supported by this forge", feature)
	}
	return err
}

func printer() *output.Printer {
	return &output.Printer{
		Format: output.ParseFormat(flagOutput),
		Out:    os.Stdout,
	}
}

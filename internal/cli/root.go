package cli

import (
	"os"

	"github.com/git-pkgs/forges/internal/config"
	"github.com/git-pkgs/forges/internal/output"
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

func printer() *output.Printer {
	return &output.Printer{
		Format: output.ParseFormat(flagOutput),
		Out:    os.Stdout,
	}
}

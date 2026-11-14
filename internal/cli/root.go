package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/config"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

var (
	flagRepo      string
	flagForgeType string
	flagHost      string
	flagOutput    string
	flagRemote    string
	flagReadOnly  bool
)

var rootCmd = &cobra.Command{
	Use:          "forge",
	Short:        "Work with git forges from the command line",
	Long:         "Supports GitHub, GitLab, Gitea, Forgejo, Bitbucket Cloud, Gerrit, and Tangled through a single interface.",
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := config.Load()
		if cfg != nil {
			if !cmd.Flags().Changed("output") && cfg.Default.Output != "" {
				flagOutput = cfg.Default.Output
			}
		}

		remote := "origin"
		if cfg != nil && cfg.Default.Remote != "" {
			remote = cfg.Default.Remote
		}
		if cmd.Flags().Changed("remote") && flagRemote != "" {
			remote = flagRemote
		}
		resolve.SetRemote(remote)
		resolve.SetHost(flagHost)
		resolve.SetForgeType(flagForgeType)

		readOnly := flagReadOnly
		if v := os.Getenv("FORGE_READ_ONLY"); v != "" && !cmd.Flags().Changed("read-only") {
			b, err := strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("invalid FORGE_READ_ONLY value %q: use true or false", v)
			}
			readOnly = b
		}
		resolve.SetReadOnly(readOnly)
		return nil
	},
}

func Execute() error {
	resolve.SetUserAgent("forge/" + Version)
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagRepo, "repo", "R", "", "Select a repository (OWNER/REPO, HOST/OWNER/REPO or rad:<RID>)")
	rootCmd.PersistentFlags().StringVar(&flagForgeType, "forge-type", "", "Force forge type: github, gitlab, gitea, forgejo, bitbucket, gerrit, tangled, radicle")
	rootCmd.PersistentFlags().StringVar(&flagHost, "host", "", "Force forge host (e.g. gitea.com, http://forgejo.local:3000); overrides FORGE_HOST and remote detection")
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "table", "Output format: table, json, plain")
	rootCmd.PersistentFlags().StringVar(&flagRemote, "remote", "", "Git remote to use when --repo (-R) is not specified (default from config, otherwise origin)")
	rootCmd.PersistentFlags().BoolVar(&flagReadOnly, "read-only", false, "Refuse API requests that could modify the forge (also FORGE_READ_ONLY=true)")
}

// notSupported wraps ErrNotSupported with a user-friendly message
// describing which feature isn't available.
func notSupported(err error, feature string) error {
	if errors.Is(err, forges.ErrNotSupported) {
		return fmt.Errorf("%s is not supported by this forge", feature)
	}
	return err
}

// repoDisplayName formats a repository for prompts and messages. Forges
// without owners, such as Radicle, show just the repository ID.
func repoDisplayName(owner, repo string) string {
	if owner == "" {
		return repo
	}
	return owner + "/" + repo
}

func printer() *output.Printer {
	return &output.Printer{
		Format: output.ParseFormat(flagOutput),
		Out:    os.Stdout,
	}
}

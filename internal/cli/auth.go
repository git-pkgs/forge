package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/git-pkgs/forge/internal/config"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd())
	authCmd.AddCommand(authStatusCmd())
}

func authLoginCmd() *cobra.Command {
	var (
		domain    string
		token     string
		forgeType string
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store credentials for a forge domain",
		RunE: func(cmd *cobra.Command, args []string) error {
			interactive := term.IsTerminal(int(os.Stdin.Fd()))
			reader := bufio.NewReader(os.Stdin)

			if domain == "" {
				if !interactive {
					return fmt.Errorf("--domain is required in non-interactive mode")
				}
				_, _ = fmt.Fprint(os.Stderr, "Domain (default: github.com): ")
				line, _ := reader.ReadString('\n')
				domain = strings.TrimSpace(line)
				if domain == "" {
					domain = "github.com"
				}
			}

			if token == "" {
				if !interactive {
					return fmt.Errorf("--token is required in non-interactive mode")
				}
				_, _ = fmt.Fprintf(os.Stderr, "Token for %s: ", domain)
				raw, err := term.ReadPassword(int(os.Stdin.Fd()))
				_, _ = fmt.Fprintln(os.Stderr) // newline after hidden input
				if err != nil {
					return fmt.Errorf("reading token: %w", err)
				}
				token = strings.TrimSpace(string(raw))
				if token == "" {
					return fmt.Errorf("token cannot be empty")
				}
			}

			if err := config.SetDomain(domain, token, forgeType); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stderr, "Stored credentials for %s\n", domain)
			return nil
		},
	}

	cmd.Flags().StringVar(&domain, "domain", "", "Forge domain (e.g. github.com, gitea.example.com)")
	cmd.Flags().StringVar(&token, "token", "", "API token")
	cmd.Flags().StringVar(&forgeType, "type", "", "Forge type: github, gitlab, gitea, forgejo")
	return cmd
}

func authStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show configured forge domains",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// Known domains to check for env var tokens
			knownDomains := []string{"github.com", "gitlab.com", "codeberg.org", "bitbucket.org"}

			// Collect all unique domains
			domains := make(map[string]bool)
			for _, d := range knownDomains {
				domains[d] = true
			}
			for d := range cfg.Domains {
				domains[d] = true
			}

			for d := range domains {
				envToken := resolve.TokenForDomainEnv(d)
				cfgSection := cfg.Domains[d]

				var sources []string
				if envToken != "" {
					sources = append(sources, "env")
				}
				if cfgSection.Token != "" {
					sources = append(sources, "config")
				}

				status := "no token"
				if len(sources) > 0 {
					status = "token from " + strings.Join(sources, ", ")
				}

				forgeType := cfgSection.Type
				if forgeType != "" {
					_, _ = fmt.Fprintf(os.Stdout, "%s (%s): %s\n", d, forgeType, status)
				} else {
					_, _ = fmt.Fprintf(os.Stdout, "%s: %s\n", d, status)
				}
			}

			return nil
		},
	}
}

package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
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
				var err error
				token, err = readTokenInteractive(domain)
				if err != nil {
					return fmt.Errorf("reading token: %w", err)
				}
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
	cmd.Flags().StringVar(&forgeType, "type", "", "Forge type: github, gitlab, gitea, forgejo, bitbucket")
	return cmd
}

// readTokenInteractive prompts for a token in raw mode.
// Pressing Ctrl+E as the first key switches to command mode (stored as "!cmd").
func readTokenInteractive(domain string) (string, error) {
	const ctrlE = 0x05

	fd := int(os.Stdin.Fd())
	_, _ = fmt.Fprintf(os.Stderr, "Token for %s (Ctrl+E first for command): ", domain)

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", fmt.Errorf("setting raw mode: %w", err)
	}

	ch, err := readOneByte(os.Stdin)
	if err != nil {
		_ = term.Restore(fd, oldState)
		_, _ = fmt.Fprintln(os.Stderr)
		return "", err
	}

	if ch == ctrlE {
		_ = term.Restore(fd, oldState)
		_, _ = fmt.Fprintln(os.Stderr)
		return readCommandInteractive(domain)
	}

	r := io.MultiReader(bytes.NewReader([]byte{ch}), os.Stdin)
	return readRawToken(fd, oldState, r)
}

func readOneByte(r io.Reader) (byte, error) {
	b := make([]byte, 1)
	_, err := r.Read(b)
	return b[0], err
}

// readRawToken accumulates a token character by character in raw mode.
// Always restores the terminal before returning.
func readRawToken(fd int, oldState *term.State, r io.Reader) (string, error) {
	const (
		ctrlC     = 0x03
		ctrlD     = 0x04
		enter     = 0x0D
		newline   = 0x0A
		backspace = 0x7F
		del       = 0x08
		printable = 0x20
	)
	defer func() {
		_ = term.Restore(fd, oldState)
		_, _ = fmt.Fprintln(os.Stderr)
	}()

	var buf []byte
	b := make([]byte, 1)
	for {
		if _, err := r.Read(b); err != nil {
			return "", err
		}

		switch b[0] {
		case ctrlC, ctrlD:
			return "", fmt.Errorf("interrupted")
		case enter, newline:
			return strings.TrimSpace(string(buf)), nil
		case backspace, del:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
		default:
			if b[0] >= printable {
				buf = append(buf, b[0])
			}
		}
	}
}

// readCommandInteractive prompts the user to enter a shell command
// whose output will be used as the token at runtime.
// Returns the command prefixed with "!" for storage in the config.
func readCommandInteractive(domain string) (string, error) {
	_, _ = fmt.Fprintf(os.Stderr, "Command for token (e.g. rbw get %s): ", domain)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	cmd := strings.TrimSpace(line)
	if cmd == "" {
		return "", fmt.Errorf("command cannot be empty")
	}
	return "!" + cmd, nil
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
					if cfgSection.TokenExec != "" {
						sources = append(sources, fmt.Sprintf("config (cmd: %s)", cfgSection.TokenExec))
					} else {
						sources = append(sources, "config")
					}
				}

				status := "no token"
				if len(sources) > 0 {
					status = "token from " + strings.Join(sources, ", ")
				}

				forgeType := cfgSection.Type
				if forgeType != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s (%s): %s\n", d, forgeType, status)
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", d, status)
				}
			}

			return nil
		},
	}
}

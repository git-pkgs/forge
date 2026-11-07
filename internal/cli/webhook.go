package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage repository webhooks",
}

func init() {
	rootCmd.AddCommand(webhookCmd)
	webhookCmd.AddCommand(webhookListCmd())
	webhookCmd.AddCommand(webhookAddCmd())
	webhookCmd.AddCommand(webhookDeleteCmd())
}

func webhookListCmd() *cobra.Command {
	var flagLimit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List webhooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.ListWebhookOpts{
				Limit: flagLimit,
			}

			hooks, err := forge.Webhooks().List(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return notSupported(err, "webhooks")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(hooks)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(hooks))
				for i, h := range hooks {
					lines[i] = fmt.Sprintf("%d\t%s", h.ID, h.URL)
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"ID", "URL", "EVENTS", "ACTIVE"}
			rows := make([][]string, len(hooks))
			for i, h := range hooks {
				rows[i] = []string{
					fmt.Sprintf("%d", h.ID),
					h.URL,
					strings.Join(h.Events, ","),
					fmt.Sprintf("%v", h.Active),
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 0, "Maximum number of webhooks")
	return cmd
}

func webhookAddCmd() *cobra.Command {
	var (
		flagURL         string
		flagSecret      string
		flagEvents      []string
		flagContentType string
		flagInsecureSSL bool
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a webhook",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagURL == "" {
				return fmt.Errorf("--url is required")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			hook, err := forge.Webhooks().Create(cmd.Context(), owner, repoName, forges.CreateWebhookOpts{
				URL:         flagURL,
				Secret:      flagSecret,
				Events:      flagEvents,
				ContentType: flagContentType,
				InsecureSSL: flagInsecureSSL,
			})
			if err != nil {
				return notSupported(err, "webhooks")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(hook)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%d %s\n", hook.ID, hook.URL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagURL, "url", "u", "", "Payload delivery URL")
	cmd.Flags().StringVarP(&flagSecret, "secret", "s", "", "Shared secret for signing deliveries")
	cmd.Flags().StringSliceVarP(&flagEvents, "event", "e", nil, "Event to subscribe to (default push)")
	cmd.Flags().StringVar(&flagContentType, "content-type", "json", "Payload format: json or form")
	cmd.Flags().BoolVar(&flagInsecureSSL, "insecure-ssl", false, "Skip TLS certificate verification")
	return cmd
}

func webhookDeleteCmd() *cobra.Command {
	var flagYes bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil || id <= 0 {
				return fmt.Errorf("invalid webhook ID: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if !flagYes {
				if err := confirm(fmt.Sprintf("Delete webhook %d in %s? [y/N] ", id, repoDisplayName(owner, repoName))); err != nil {
					return err
				}
			}

			if err := forge.Webhooks().Delete(cmd.Context(), owner, repoName, id); err != nil {
				return notSupported(err, "webhooks")
			}

			_, _ = fmt.Fprintf(os.Stdout, "Deleted webhook %d\n", id)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation")
	return cmd
}

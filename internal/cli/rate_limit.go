package cli

import (
	"fmt"
	"time"

	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

var rateLimitCmd = &cobra.Command{
	Use:   "rate-limit",
	Short: "Check API rate limit status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := domainFromFlags()
		f, err := resolve.ForgeForDomain(domain)
		if err != nil {
			return err
		}

		rl, err := f.GetRateLimit(cmd.Context())
		if err != nil {
			return notSupported(err, "rate limit")
		}

		p := printer()
		if p.Format == output.JSON {
			return p.PrintJSON(rl)
		}

		if p.Format == output.Plain {
			p.PrintPlain([]string{
				fmt.Sprintf("%d\t%d\t%s", rl.Limit, rl.Remaining, rl.Reset.Format(time.RFC3339)),
			})
			return nil
		}

		headers := []string{"LIMIT", "REMAINING", "RESETS AT"}
		rows := [][]string{
			{
				fmt.Sprintf("%d", rl.Limit),
				fmt.Sprintf("%d", rl.Remaining),
				formatReset(rl.Reset),
			},
		}
		p.PrintTable(headers, rows)
		return nil
	},
}

func formatReset(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	until := time.Until(t).Round(time.Second)
	if until <= 0 {
		return "now"
	}
	return fmt.Sprintf("%s (%s)", t.Format(time.RFC3339), until)
}

func init() {
	rootCmd.AddCommand(rateLimitCmd)
}

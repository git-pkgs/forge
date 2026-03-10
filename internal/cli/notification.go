package cli

import (
	"fmt"
	"os"

	forges "github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

var notificationCmd = &cobra.Command{
	Use:     "notification",
	Short:   "Manage notifications",
	Aliases: []string{"notif"},
}

func init() {
	rootCmd.AddCommand(notificationCmd)
	notificationCmd.AddCommand(notificationListCmd())
	notificationCmd.AddCommand(notificationReadCmd())
}

func notificationListCmd() *cobra.Command {
	var (
		flagUnread bool
		flagNRepo  string
		flagLimit  int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List notifications",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := domainFromFlags()
			f, err := resolve.ForgeForDomain(domain)
			if err != nil {
				return err
			}

			notifications, err := f.Notifications().List(cmd.Context(), forges.ListNotificationOpts{
				Repo:   flagNRepo,
				Unread: flagUnread,
				Limit:  flagLimit,
			})
			if err != nil {
				return notSupported(err, "notifications")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(notifications)
			}

			if len(notifications) == 0 {
				_, _ = fmt.Fprintln(os.Stdout, "No notifications")
				return nil
			}

			if p.Format == output.Plain {
				lines := make([]string, len(notifications))
				for i, n := range notifications {
					status := "read"
					if n.Unread {
						status = "unread"
					}
					lines[i] = fmt.Sprintf("%s\t%s\t%s\t%s", n.ID, status, n.Repo, n.Title)
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"ID", "STATUS", "TYPE", "REPO", "TITLE"}
			rows := make([][]string, len(notifications))
			for i, n := range notifications {
				status := "read"
				if n.Unread {
					status = "unread"
				}
				title := n.Title
				if len(title) > 60 {
					title = title[:57] + "..."
				}
				rows[i] = []string{
					n.ID,
					status,
					string(n.SubjectType),
					n.Repo,
					title,
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagUnread, "unread", false, "Only show unread notifications")
	cmd.Flags().StringVar(&flagNRepo, "repo", "", "Filter by repository (owner/repo)")
	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 30, "Maximum number of notifications")
	return cmd
}

func notificationReadCmd() *cobra.Command {
	var (
		flagID    string
		flagNRepo string
	)

	cmd := &cobra.Command{
		Use:   "read",
		Short: "Mark notifications as read",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := domainFromFlags()
			f, err := resolve.ForgeForDomain(domain)
			if err != nil {
				return err
			}

			err = f.Notifications().MarkRead(cmd.Context(), forges.MarkNotificationOpts{
				ID:   flagID,
				Repo: flagNRepo,
			})
			if err != nil {
				return notSupported(err, "marking notifications as read")
			}

			what := "all notifications"
			if flagID != "" {
				what = "notification " + flagID
			} else if flagNRepo != "" {
				what = "notifications for " + flagNRepo
			}
			_, _ = fmt.Fprintf(os.Stdout, "Marked %s as read\n", what)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagID, "id", "", "Mark a specific notification thread as read")
	cmd.Flags().StringVar(&flagNRepo, "repo", "", "Mark all notifications in a repository as read")
	return cmd
}

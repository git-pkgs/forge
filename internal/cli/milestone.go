package cli

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

const defaultMilestoneLimit = 30

var milestoneCmd = &cobra.Command{
	Use:   "milestone",
	Short: "Manage milestones",
}

func init() {
	rootCmd.AddCommand(milestoneCmd)
	milestoneCmd.AddCommand(milestoneListCmd())
	milestoneCmd.AddCommand(milestoneViewCmd())
	milestoneCmd.AddCommand(milestoneCreateCmd())
	milestoneCmd.AddCommand(milestoneEditCmd())
	milestoneCmd.AddCommand(milestoneCloseCmd())
	milestoneCmd.AddCommand(milestoneReopenCmd())
	milestoneCmd.AddCommand(milestoneDeleteCmd())
}

func milestoneListCmd() *cobra.Command {
	var (
		flagState string
		flagLimit int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List milestones",
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.ListMilestoneOpts{
				State: flagState,
				Limit: flagLimit,
			}

			milestones, err := forge.Milestones().List(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return notSupported(err, "milestones")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(milestones)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(milestones))
				for i, m := range milestones {
					lines[i] = fmt.Sprintf("%d\t%s", m.Number, m.Title)
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"#", "TITLE", "STATE", "DUE"}
			rows := make([][]string, len(milestones))
			for i, m := range milestones {
				due := ""
				if m.DueDate != nil {
					due = m.DueDate.Format("2006-01-02")
				}
				rows[i] = []string{
					strconv.Itoa(m.Number),
					m.Title,
					m.State,
					due,
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagState, "state", "s", "open", "Filter by state: open, closed, all")
	cmd.Flags().IntVarP(&flagLimit, "limit", "L", defaultMilestoneLimit, "Maximum number of milestones")
	return cmd
}

func milestoneViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <number>",
		Short: "View a milestone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid milestone number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			milestone, err := forge.Milestones().Get(cmd.Context(), owner, repoName, id)
			if err != nil {
				return notSupported(err, "milestones")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(milestone)
			}

			_, _ = fmt.Fprintf(os.Stdout, "#%d %s\n", milestone.Number, milestone.Title)
			_, _ = fmt.Fprintf(os.Stdout, "State:  %s\n", milestone.State)
			if milestone.DueDate != nil {
				_, _ = fmt.Fprintf(os.Stdout, "Due:    %s\n", milestone.DueDate.Format("2006-01-02"))
			}
			if milestone.Description != "" {
				_, _ = fmt.Fprintln(os.Stdout)
				_, _ = fmt.Fprintln(os.Stdout, milestone.Description)
			}

			return nil
		},
	}
}

func milestoneCreateCmd() *cobra.Command {
	var (
		flagTitle       string
		flagDescription string
		flagDueDate     string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a milestone",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagTitle == "" {
				return fmt.Errorf("--title is required")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.CreateMilestoneOpts{
				Title:       flagTitle,
				Description: flagDescription,
			}

			if flagDueDate != "" {
				t, err := time.Parse("2006-01-02", flagDueDate)
				if err != nil {
					return fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)
				}
				opts.DueDate = &t
			}

			milestone, err := forge.Milestones().Create(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return notSupported(err, "milestones")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(milestone)
			}

			_, _ = fmt.Fprintf(os.Stdout, "#%d %s\n", milestone.Number, milestone.Title)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagTitle, "title", "t", "", "Milestone title")
	cmd.Flags().StringVarP(&flagDescription, "description", "d", "", "Milestone description")
	cmd.Flags().StringVar(&flagDueDate, "due", "", "Due date (YYYY-MM-DD)")
	return cmd
}

func milestoneEditCmd() *cobra.Command {
	var (
		flagTitle       string
		flagDescription string
		flagDueDate     string
	)

	cmd := &cobra.Command{
		Use:   "edit <number>",
		Short: "Edit a milestone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid milestone number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.UpdateMilestoneOpts{}
			if cmd.Flags().Changed("title") {
				opts.Title = &flagTitle
			}
			if cmd.Flags().Changed("description") {
				opts.Description = &flagDescription
			}
			if cmd.Flags().Changed("due") {
				t, err := time.Parse("2006-01-02", flagDueDate)
				if err != nil {
					return fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)
				}
				opts.DueDate = &t
			}

			milestone, err := forge.Milestones().Update(cmd.Context(), owner, repoName, id, opts)
			if err != nil {
				return notSupported(err, "milestones")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(milestone)
			}

			_, _ = fmt.Fprintf(os.Stdout, "#%d %s\n", milestone.Number, milestone.Title)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagTitle, "title", "t", "", "Set the title")
	cmd.Flags().StringVarP(&flagDescription, "description", "d", "", "Set the description")
	cmd.Flags().StringVar(&flagDueDate, "due", "", "Set the due date (YYYY-MM-DD)")
	return cmd
}

func milestoneCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close <number>",
		Short: "Close a milestone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid milestone number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if err := forge.Milestones().Close(cmd.Context(), owner, repoName, id); err != nil {
				return notSupported(err, "milestones")
			}

			_, _ = fmt.Fprintf(os.Stdout, "Closed #%d\n", id)
			return nil
		},
	}
}

func milestoneReopenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reopen <number>",
		Short: "Reopen a milestone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid milestone number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if err := forge.Milestones().Reopen(cmd.Context(), owner, repoName, id); err != nil {
				return notSupported(err, "milestones")
			}

			_, _ = fmt.Fprintf(os.Stdout, "Reopened #%d\n", id)
			return nil
		},
	}
}

func milestoneDeleteCmd() *cobra.Command {
	var flagYes bool

	cmd := &cobra.Command{
		Use:   "delete <number>",
		Short: "Delete a milestone",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid milestone number: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if !flagYes {
				if err := confirm(fmt.Sprintf("Delete milestone #%d in %s/%s? [y/N] ", id, owner, repoName)); err != nil {
					return err
				}
			}

			if err := forge.Milestones().Delete(cmd.Context(), owner, repoName, id); err != nil {
				return notSupported(err, "milestones")
			}

			_, _ = fmt.Fprintf(os.Stdout, "Deleted #%d\n", id)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation")
	return cmd
}

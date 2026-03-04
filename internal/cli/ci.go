package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/git-pkgs/forges/internal/output"
	"github.com/git-pkgs/forges/internal/resolve"
	"github.com/git-pkgs/forges"
	"github.com/spf13/cobra"
)

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Manage CI/CD pipelines",
}

func init() {
	rootCmd.AddCommand(ciCmd)
	ciCmd.AddCommand(ciListCmd())
	ciCmd.AddCommand(ciViewCmd())
	ciCmd.AddCommand(ciRunCmd())
	ciCmd.AddCommand(ciCancelCmd())
	ciCmd.AddCommand(ciRetryCmd())
	ciCmd.AddCommand(ciLogCmd())
}

func ciListCmd() *cobra.Command {
	var (
		flagBranch   string
		flagStatus   string
		flagUser     string
		flagWorkflow string
		flagLimit    int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pipeline runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.ListCIRunOpts{
				Branch:   flagBranch,
				Status:   flagStatus,
				User:     flagUser,
				Workflow: flagWorkflow,
				Limit:    flagLimit,
			}

			runs, err := forge.CI().ListRuns(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(runs)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(runs))
				for i, r := range runs {
					lines[i] = fmt.Sprintf("%d\t%s\t%s", r.ID, r.Title, r.Status)
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"ID", "TITLE", "STATUS", "BRANCH", "EVENT", "CREATED"}
			rows := make([][]string, len(runs))
			for i, r := range runs {
				status := r.Status
				if r.Conclusion != "" {
					status = r.Conclusion
				}
				created := ""
				if !r.CreatedAt.IsZero() {
					created = r.CreatedAt.Format("2006-01-02 15:04")
				}
				rows[i] = []string{
					strconv.FormatInt(r.ID, 10),
					r.Title,
					status,
					r.Branch,
					r.Event,
					created,
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagBranch, "branch", "b", "", "Filter by branch")
	cmd.Flags().StringVarP(&flagStatus, "status", "s", "", "Filter by status")
	cmd.Flags().StringVarP(&flagUser, "user", "u", "", "Filter by user")
	cmd.Flags().StringVar(&flagWorkflow, "workflow", "", "Filter by workflow name or file")
	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 20, "Maximum number of runs")
	return cmd
}

func ciViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <run-id>",
		Short: "View a pipeline run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid run ID: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			run, err := forge.CI().GetRun(cmd.Context(), owner, repoName, runID)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(run)
			}

			status := run.Status
			if run.Conclusion != "" {
				status = run.Conclusion
			}

			fmt.Fprintf(os.Stdout, "#%d %s\n", run.ID, run.Title)
			fmt.Fprintf(os.Stdout, "Status:  %s\n", status)
			fmt.Fprintf(os.Stdout, "Branch:  %s\n", run.Branch)
			if run.SHA != "" {
				fmt.Fprintf(os.Stdout, "SHA:     %s\n", run.SHA[:min(7, len(run.SHA))])
			}
			if run.HTMLURL != "" {
				fmt.Fprintf(os.Stdout, "URL:     %s\n", run.HTMLURL)
			}

			if len(run.Jobs) > 0 {
				fmt.Fprintln(os.Stdout)
				fmt.Fprintln(os.Stdout, "Jobs:")
				for _, j := range run.Jobs {
					jStatus := j.Status
					if j.Conclusion != "" {
						jStatus = j.Conclusion
					}
					fmt.Fprintf(os.Stdout, "  %s  %s\n", j.Name, jStatus)
				}
			}

			return nil
		},
	}
}

func ciRunCmd() *cobra.Command {
	var (
		flagBranch string
		flagFields []string
	)

	cmd := &cobra.Command{
		Use:   "run [<workflow>]",
		Short: "Trigger a workflow run",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.TriggerCIRunOpts{
				Branch: flagBranch,
			}
			if len(args) > 0 {
				opts.Workflow = args[0]
			}

			if len(flagFields) > 0 {
				opts.Inputs = make(map[string]string)
				for _, f := range flagFields {
					k, v, ok := strings.Cut(f, "=")
					if !ok {
						return fmt.Errorf("invalid field format: %q (expected KEY=VALUE)", f)
					}
					opts.Inputs[k] = v
				}
			}

			if err := forge.CI().TriggerRun(cmd.Context(), owner, repoName, opts); err != nil {
				return err
			}

			fmt.Fprintln(os.Stdout, "Triggered workflow run")
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagBranch, "branch", "b", "", "Branch to run on")
	cmd.Flags().StringArrayVarP(&flagFields, "field", "F", nil, "Input parameters (KEY=VALUE)")
	return cmd
}

func ciCancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <run-id>",
		Short: "Cancel a running pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid run ID: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if err := forge.CI().CancelRun(cmd.Context(), owner, repoName, runID); err != nil {
				return err
			}

			fmt.Fprintf(os.Stdout, "Cancelled run %d\n", runID)
			return nil
		},
	}
}

func ciRetryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "retry <run-id>",
		Short: "Retry a failed pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid run ID: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if err := forge.CI().RetryRun(cmd.Context(), owner, repoName, runID); err != nil {
				return err
			}

			fmt.Fprintf(os.Stdout, "Retried run %d\n", runID)
			return nil
		},
	}
}

func ciLogCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "log <job-id>",
		Short: "View job logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid job ID: %s", args[0])
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			rc, err := forge.CI().GetJobLog(cmd.Context(), owner, repoName, jobID)
			if err != nil {
				return err
			}
			defer rc.Close()

			_, err = io.Copy(os.Stdout, rc)
			return err
		},
	}
}

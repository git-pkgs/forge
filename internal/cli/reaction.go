package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

func issueReactionsCmd() *cobra.Command {
	var flagComment int64

	cmd := &cobra.Command{
		Use:   "reactions <number>",
		Short: "List reactions on an issue comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}
			if flagComment == 0 {
				return fmt.Errorf("--comment is required")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			reactions, err := forge.Issues().ListReactions(cmd.Context(), owner, repoName, number, flagComment)
			if err != nil {
				return notSupported(err, "comment reactions")
			}

			return printReactions(reactions)
		},
	}

	cmd.Flags().Int64Var(&flagComment, "comment", 0, "Comment ID")
	return cmd
}

func issueReactCmd() *cobra.Command {
	var (
		flagComment  int64
		flagReaction string
	)

	cmd := &cobra.Command{
		Use:   "react <number>",
		Short: "Add a reaction to an issue comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}
			if flagComment == 0 {
				return fmt.Errorf("--comment is required")
			}
			if flagReaction == "" {
				return fmt.Errorf("--reaction is required")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			reaction, err := forge.Issues().AddReaction(cmd.Context(), owner, repoName, number, flagComment, flagReaction)
			if err != nil {
				return notSupported(err, "comment reactions")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(reaction)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Added %s reaction\n", reaction.Content)
			return nil
		},
	}

	cmd.Flags().Int64Var(&flagComment, "comment", 0, "Comment ID")
	cmd.Flags().StringVar(&flagReaction, "reaction", "", "Reaction to add (e.g. +1, -1, laugh, hooray, confused, heart, rocket, eyes)")
	return cmd
}

func prReactionsCmd() *cobra.Command {
	var flagComment int64

	cmd := &cobra.Command{
		Use:   "reactions <number>",
		Short: "List reactions on a pull request comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}
			if flagComment == 0 {
				return fmt.Errorf("--comment is required")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			reactions, err := forge.PullRequests().ListReactions(cmd.Context(), owner, repoName, number, flagComment)
			if err != nil {
				return notSupported(err, "comment reactions")
			}

			return printReactions(reactions)
		},
	}

	cmd.Flags().Int64Var(&flagComment, "comment", 0, "Comment ID")
	return cmd
}

func prReactCmd() *cobra.Command {
	var (
		flagComment  int64
		flagReaction string
	)

	cmd := &cobra.Command{
		Use:   "react <number>",
		Short: "Add a reaction to a pull request comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}
			if flagComment == 0 {
				return fmt.Errorf("--comment is required")
			}
			if flagReaction == "" {
				return fmt.Errorf("--reaction is required")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			reaction, err := forge.PullRequests().AddReaction(cmd.Context(), owner, repoName, number, flagComment, flagReaction)
			if err != nil {
				return notSupported(err, "comment reactions")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(reaction)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Added %s reaction\n", reaction.Content)
			return nil
		},
	}

	cmd.Flags().Int64Var(&flagComment, "comment", 0, "Comment ID")
	cmd.Flags().StringVar(&flagReaction, "reaction", "", "Reaction to add (e.g. +1, -1, laugh, hooray, confused, heart, rocket, eyes)")
	return cmd
}

func printReactions(reactions []forges.Reaction) error {
	p := printer()
	if p.Format == output.JSON {
		return p.PrintJSON(reactions)
	}

	if len(reactions) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No reactions")
		return nil
	}

	if p.Format == output.Plain {
		lines := make([]string, len(reactions))
		for i, r := range reactions {
			lines[i] = fmt.Sprintf("%s\t%s", r.Content, r.User)
		}
		p.PrintPlain(lines)
		return nil
	}

	headers := []string{"REACTION", "USER"}
	rows := make([][]string, len(reactions))
	for i, r := range reactions {
		rows[i] = []string{r.Content, r.User}
	}
	p.PrintTable(headers, rows)
	return nil
}

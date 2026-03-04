package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/git-pkgs/forge"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories",
}

func init() {
	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoViewCmd())
	repoCmd.AddCommand(repoListCmd())
	repoCmd.AddCommand(repoCreateCmd())
	repoCmd.AddCommand(repoEditCmd())
	repoCmd.AddCommand(repoDeleteCmd())
	repoCmd.AddCommand(repoForkCmd())
	repoCmd.AddCommand(repoCloneCmd())
	repoCmd.AddCommand(repoSearchCmd())
}

func repoViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view [OWNER/REPO]",
		Short: "View repository details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := flagRepo
			if len(args) > 0 {
				repo = args[0]
			}

			var overrideFlag string
			if repo != "" {
				overrideFlag = repo
			}

			forge, owner, repoName, _, err := resolve.Repo(overrideFlag, flagForgeType)
			if err != nil {
				return err
			}

			r, err := forge.Repos().Get(cmd.Context(), owner, repoName)
			if err != nil {
				return fmt.Errorf("getting repo %s/%s: %w", owner, repoName, err)
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(r)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s\n", r.FullName)
			if r.Description != "" {
				_, _ = fmt.Fprintf(os.Stdout, "%s\n", r.Description)
			}
			_, _ = fmt.Fprintln(os.Stdout)

			if r.HTMLURL != "" {
				_, _ = fmt.Fprintf(os.Stdout, "URL:       %s\n", r.HTMLURL)
			}
			if r.Language != "" {
				_, _ = fmt.Fprintf(os.Stdout, "Language:  %s\n", r.Language)
			}
			if r.License != "" {
				_, _ = fmt.Fprintf(os.Stdout, "License:   %s\n", r.License)
			}
			if r.DefaultBranch != "" {
				_, _ = fmt.Fprintf(os.Stdout, "Branch:    %s\n", r.DefaultBranch)
			}

			var flags []string
			if r.Private {
				flags = append(flags, "private")
			}
			if r.Fork {
				flags = append(flags, "fork")
			}
			if r.Archived {
				flags = append(flags, "archived")
			}
			if len(flags) > 0 {
				_, _ = fmt.Fprintf(os.Stdout, "Flags:     %s\n", strings.Join(flags, ", "))
			}

			_, _ = fmt.Fprintf(os.Stdout, "Stars:     %d\n", r.StargazersCount)
			_, _ = fmt.Fprintf(os.Stdout, "Forks:     %d\n", r.ForksCount)
			_, _ = fmt.Fprintf(os.Stdout, "Issues:    %d\n", r.OpenIssuesCount)

			return nil
		},
	}
}

func repoListCmd() *cobra.Command {
	var (
		flagLimit          int
		flagNoArchived     bool
		flagNoForks        bool
		flagArchivedOnly   bool
		flagForksOnly      bool
	)

	cmd := &cobra.Command{
		Use:   "list <owner>",
		Short: "List repositories",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner := args[0]

			domain := domainFromFlags()
			forge, err := resolve.ForgeForDomain(domain)
			if err != nil {
				return err
			}

			opts := forges.ListRepoOpts{
				PerPage: flagLimit,
			}
			if flagNoArchived {
				opts.Archived = forges.ArchivedExclude
			}
			if flagArchivedOnly {
				opts.Archived = forges.ArchivedOnly
			}
			if flagNoForks {
				opts.Forks = forges.ForkExclude
			}
			if flagForksOnly {
				opts.Forks = forges.ForkOnly
			}

			repos, err := forge.Repos().List(cmd.Context(), owner, opts)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(repos)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(repos))
				for i, r := range repos {
					lines[i] = r.FullName
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"NAME", "DESCRIPTION", "LANGUAGE", "STARS"}
			rows := make([][]string, len(repos))
			for i, r := range repos {
				desc := r.Description
				if len(desc) > 60 {
					desc = desc[:57] + "..."
				}
				rows[i] = []string{
					r.FullName,
					desc,
					r.Language,
					strconv.Itoa(r.StargazersCount),
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 100, "Maximum number of repos to return")
	cmd.Flags().BoolVar(&flagNoArchived, "no-archived", false, "Exclude archived repos")
	cmd.Flags().BoolVar(&flagNoForks, "no-forks", false, "Exclude forked repos")
	cmd.Flags().BoolVar(&flagArchivedOnly, "archived", false, "Only show archived repos")
	cmd.Flags().BoolVar(&flagForksOnly, "forks", false, "Only show forked repos")
	return cmd
}

func repoCreateCmd() *cobra.Command {
	var (
		flagDescription   string
		flagPrivate       bool
		flagPublic        bool
		flagInternal      bool
		flagClone         bool
		flagInit          bool
		flagReadme        bool
		flagGitignore     string
		flagLicense       string
		flagDefaultBranch string
		flagOwner         string
	)

	cmd := &cobra.Command{
		Use:   "create [<name>]",
		Short: "Create a new repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			domain := domainFromFlags()
			forge, err := resolve.ForgeForDomain(domain)
			if err != nil {
				return err
			}

			opts := forges.CreateRepoOpts{
				Name:          name,
				Description:   flagDescription,
				Init:          flagInit,
				Readme:        flagReadme,
				Gitignore:     flagGitignore,
				License:       flagLicense,
				DefaultBranch: flagDefaultBranch,
				Owner:         flagOwner,
			}

			if flagPrivate {
				opts.Visibility = forges.VisibilityPrivate
			} else if flagInternal {
				opts.Visibility = forges.VisibilityInternal
			} else if flagPublic {
				opts.Visibility = forges.VisibilityPublic
			}

			repo, err := forge.Repos().Create(cmd.Context(), opts)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(repo)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s\n", repo.HTMLURL)

			if flagClone && repo.CloneURL != "" {
				cloneCmd := exec.CommandContext(cmd.Context(), "git", "clone", repo.CloneURL)
				cloneCmd.Stdout = os.Stdout
				cloneCmd.Stderr = os.Stderr
				return cloneCmd.Run()
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&flagDescription, "description", "d", "", "Repository description")
	cmd.Flags().BoolVar(&flagPrivate, "private", false, "Make private")
	cmd.Flags().BoolVar(&flagPublic, "public", false, "Make public")
	cmd.Flags().BoolVar(&flagInternal, "internal", false, "Make internal")
	cmd.Flags().BoolVarP(&flagClone, "clone", "c", false, "Clone after creation")
	cmd.Flags().BoolVar(&flagInit, "init", false, "Initialize with default branch")
	cmd.Flags().BoolVar(&flagReadme, "readme", false, "Add a README")
	cmd.Flags().StringVarP(&flagGitignore, "gitignore", "g", "", "Gitignore template")
	cmd.Flags().StringVar(&flagLicense, "license", "", "License template")
	cmd.Flags().StringVar(&flagDefaultBranch, "default-branch", "", "Default branch name")
	cmd.Flags().StringVar(&flagOwner, "owner", "", "Owner or group")
	return cmd
}

func repoEditCmd() *cobra.Command {
	var (
		flagDescription   string
		flagHomepage      string
		flagDefaultBranch string
		flagPrivate       bool
		flagPublic        bool
	)

	cmd := &cobra.Command{
		Use:   "edit [OWNER/REPO]",
		Short: "Edit repository settings",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := flagRepo
			if len(args) > 0 {
				repo = args[0]
			}

			var overrideFlag string
			if repo != "" {
				overrideFlag = repo
			}

			forge, owner, repoName, _, err := resolve.Repo(overrideFlag, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.EditRepoOpts{}
			if cmd.Flags().Changed("description") {
				opts.Description = &flagDescription
			}
			if cmd.Flags().Changed("homepage") {
				opts.Homepage = &flagHomepage
			}
			if cmd.Flags().Changed("default-branch") {
				opts.DefaultBranch = &flagDefaultBranch
			}
			if flagPrivate {
				opts.Visibility = forges.VisibilityPrivate
			}
			if flagPublic {
				opts.Visibility = forges.VisibilityPublic
			}

			r, err := forge.Repos().Edit(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(r)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s\n", r.HTMLURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagDescription, "description", "d", "", "New description")
	cmd.Flags().StringVar(&flagHomepage, "homepage", "", "New homepage URL")
	cmd.Flags().StringVar(&flagDefaultBranch, "default-branch", "", "New default branch")
	cmd.Flags().BoolVar(&flagPrivate, "private", false, "Make private")
	cmd.Flags().BoolVar(&flagPublic, "public", false, "Make public")
	return cmd
}

func repoDeleteCmd() *cobra.Command {
	var flagYes bool

	cmd := &cobra.Command{
		Use:   "delete [OWNER/REPO]",
		Short: "Delete a repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := flagRepo
			if len(args) > 0 {
				repo = args[0]
			}

			var overrideFlag string
			if repo != "" {
				overrideFlag = repo
			}

			forge, owner, repoName, _, err := resolve.Repo(overrideFlag, flagForgeType)
			if err != nil {
				return err
			}

			if !flagYes {
				if err := confirm(fmt.Sprintf("Delete %s/%s? This cannot be undone. [y/N] ", owner, repoName)); err != nil {
					return err
				}
			}

			if err := forge.Repos().Delete(cmd.Context(), owner, repoName); err != nil {
				return fmt.Errorf("deleting repo %s/%s: %w", owner, repoName, err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Deleted %s/%s\n", owner, repoName)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func repoForkCmd() *cobra.Command {
	var (
		flagOwner string
		flagName  string
		flagClone bool
	)

	cmd := &cobra.Command{
		Use:   "fork [OWNER/REPO]",
		Short: "Fork a repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := flagRepo
			if len(args) > 0 {
				repo = args[0]
			}

			var overrideFlag string
			if repo != "" {
				overrideFlag = repo
			}

			forge, owner, repoName, _, err := resolve.Repo(overrideFlag, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.ForkRepoOpts{
				Owner: flagOwner,
				Name:  flagName,
			}

			r, err := forge.Repos().Fork(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(r)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s\n", r.HTMLURL)

			if flagClone && r.CloneURL != "" {
				cloneCmd := exec.CommandContext(cmd.Context(), "git", "clone", r.CloneURL)
				cloneCmd.Stdout = os.Stdout
				cloneCmd.Stderr = os.Stderr
				return cloneCmd.Run()
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flagOwner, "owner", "", "Target owner/org for the fork")
	cmd.Flags().StringVar(&flagName, "name", "", "Name for the fork")
	cmd.Flags().BoolVarP(&flagClone, "clone", "c", false, "Clone the fork after creation")
	return cmd
}

func repoCloneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clone <OWNER/REPO>",
		Short: "Clone a repository locally",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, owner, repoName, _, err := resolve.Repo(args[0], flagForgeType)
			if err != nil {
				return err
			}

			r, err := forge.Repos().Get(cmd.Context(), owner, repoName)
			if err != nil {
				return err
			}

			cloneURL := r.CloneURL
			if cloneURL == "" {
				cloneURL = r.HTMLURL + ".git"
			}

			cloneCmd := exec.CommandContext(cmd.Context(), "git", "clone", cloneURL)
			cloneCmd.Stdout = os.Stdout
			cloneCmd.Stderr = os.Stderr
			return cloneCmd.Run()
		},
	}
}

func repoSearchCmd() *cobra.Command {
	var (
		flagLimit int
		flagSort  string
		flagOrder string
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for repositories",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := domainFromFlags()
			forge, err := resolve.ForgeForDomain(domain)
			if err != nil {
				return err
			}

			opts := forges.SearchRepoOpts{
				Query:   args[0],
				Sort:    flagSort,
				Order:   flagOrder,
				PerPage: flagLimit,
			}

			repos, err := forge.Repos().Search(cmd.Context(), opts)
			if err != nil {
				if errors.Is(err, forges.ErrNotSupported) {
					return fmt.Errorf("search is not supported for this forge")
				}
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(repos)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(repos))
				for i, r := range repos {
					lines[i] = r.FullName
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"NAME", "DESCRIPTION", "STARS"}
			rows := make([][]string, len(repos))
			for i, r := range repos {
				desc := r.Description
				if len(desc) > 60 {
					desc = desc[:57] + "..."
				}
				rows[i] = []string{
					r.FullName,
					desc,
					strconv.Itoa(r.StargazersCount),
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().StringVar(&flagSort, "sort", "", "Sort field (stars, forks, updated)")
	cmd.Flags().StringVar(&flagOrder, "order", "", "Sort order (asc, desc)")
	return cmd
}

func domainFromFlags() string {
	return resolve.DomainFromForgeType(flagForgeType)
}

package cli

import (
	"fmt"
	"os"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

const (
	defaultReleaseLimit = 30
	releaseAssetArgs    = 2
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Manage releases",
}

func init() {
	rootCmd.AddCommand(releaseCmd)
	releaseCmd.AddCommand(releaseListCmd())
	releaseCmd.AddCommand(releaseViewCmd())
	releaseCmd.AddCommand(releaseCreateCmd())
	releaseCmd.AddCommand(releaseEditCmd())
	releaseCmd.AddCommand(releaseDeleteCmd())
	releaseCmd.AddCommand(releaseUploadCmd())
	releaseCmd.AddCommand(releaseDownloadCmd())
}

func releaseListCmd() *cobra.Command {
	var flagLimit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List releases",
		RunE: func(cmd *cobra.Command, args []string) error {
			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.ListReleaseOpts{
				Limit: flagLimit,
			}

			releases, err := forge.Releases().List(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return notSupported(err, "releases")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(releases)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(releases))
				for i, r := range releases {
					lines[i] = r.TagName
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"TAG", "TITLE", "DRAFT", "PRERELEASE", "PUBLISHED"}
			rows := make([][]string, len(releases))
			for i, r := range releases {
				published := ""
				if !r.PublishedAt.IsZero() {
					published = r.PublishedAt.Format("2006-01-02")
				}
				rows[i] = []string{
					output.Sanitize(r.TagName),
					output.Sanitize(r.Title),
					fmt.Sprintf("%v", r.Draft),
					fmt.Sprintf("%v", r.Prerelease),
					published,
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().IntVarP(&flagLimit, "limit", "L", defaultReleaseLimit, "Maximum number of releases")
	return cmd
}

func releaseViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <tag>",
		Short: "View a release",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := args[0]

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			release, err := forge.Releases().Get(cmd.Context(), owner, repoName, tag)
			if err != nil {
				return notSupported(err, "releases")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(release)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s %s\n", output.Sanitize(release.TagName), output.Sanitize(release.Title))
			if release.Draft {
				_, _ = fmt.Fprintln(os.Stdout, "Draft: true")
			}
			if release.Prerelease {
				_, _ = fmt.Fprintln(os.Stdout, "Prerelease: true")
			}
			if !release.PublishedAt.IsZero() {
				_, _ = fmt.Fprintf(os.Stdout, "Published: %s\n", release.PublishedAt.Format("2006-01-02"))
			}
			if release.HTMLURL != "" {
				_, _ = fmt.Fprintf(os.Stdout, "URL: %s\n", release.HTMLURL)
			}

			if len(release.Assets) > 0 {
				_, _ = fmt.Fprintln(os.Stdout)
				_, _ = fmt.Fprintln(os.Stdout, "Assets:")
				for _, a := range release.Assets {
					_, _ = fmt.Fprintf(os.Stdout, "  %s (%d bytes)\n", output.Sanitize(a.Name), a.Size)
				}
			}

			if release.Body != "" {
				_, _ = fmt.Fprintln(os.Stdout)
				_, _ = fmt.Fprintln(os.Stdout, output.Sanitize(release.Body))
			}

			return nil
		},
	}
}

func releaseCreateCmd() *cobra.Command {
	var (
		flagTag        string
		flagTitle      string
		flagBody       string
		flagTarget     string
		flagDraft      bool
		flagPrerelease bool
		flagNotes      bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a release",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagTag == "" {
				return fmt.Errorf("--tag is required")
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			title := flagTitle
			if title == "" {
				title = flagTag
			}

			opts := forges.CreateReleaseOpts{
				TagName:       flagTag,
				Title:         title,
				Body:          flagBody,
				Target:        flagTarget,
				Draft:         flagDraft,
				Prerelease:    flagPrerelease,
				GenerateNotes: flagNotes,
			}

			release, err := forge.Releases().Create(cmd.Context(), owner, repoName, opts)
			if err != nil {
				return notSupported(err, "releases")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(release)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s %s\n", output.Sanitize(release.TagName), output.Sanitize(release.Title))
			if release.HTMLURL != "" {
				_, _ = fmt.Fprintln(os.Stdout, release.HTMLURL)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&flagTag, "tag", "", "Tag name for the release")
	cmd.Flags().StringVarP(&flagTitle, "title", "t", "", "Release title (defaults to tag)")
	cmd.Flags().StringVarP(&flagBody, "body", "b", "", "Release description")
	cmd.Flags().StringVar(&flagTarget, "target", "", "Target branch or commit")
	cmd.Flags().BoolVar(&flagDraft, "draft", false, "Create as draft")
	cmd.Flags().BoolVar(&flagPrerelease, "prerelease", false, "Mark as prerelease")
	cmd.Flags().BoolVar(&flagNotes, "generate-notes", false, "Auto-generate release notes")
	return cmd
}

func releaseEditCmd() *cobra.Command {
	var (
		flagTitle      string
		flagBody       string
		flagTarget     string
		flagTagName    string
		flagDraft      bool
		flagPrerelease bool
	)

	cmd := &cobra.Command{
		Use:   "edit <tag>",
		Short: "Edit a release",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := args[0]

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			opts := forges.UpdateReleaseOpts{}
			if cmd.Flags().Changed("title") {
				opts.Title = &flagTitle
			}
			if cmd.Flags().Changed("body") {
				opts.Body = &flagBody
			}
			if cmd.Flags().Changed("target") {
				opts.Target = &flagTarget
			}
			if cmd.Flags().Changed("tag") {
				opts.TagName = &flagTagName
			}
			if cmd.Flags().Changed("draft") {
				opts.Draft = &flagDraft
			}
			if cmd.Flags().Changed("prerelease") {
				opts.Prerelease = &flagPrerelease
			}

			release, err := forge.Releases().Update(cmd.Context(), owner, repoName, tag, opts)
			if err != nil {
				return notSupported(err, "releases")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(release)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s %s\n", output.Sanitize(release.TagName), output.Sanitize(release.Title))
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagTitle, "title", "t", "", "Set the title")
	cmd.Flags().StringVarP(&flagBody, "body", "b", "", "Set the description")
	cmd.Flags().StringVar(&flagTarget, "target", "", "Set the target branch or commit")
	cmd.Flags().StringVar(&flagTagName, "tag", "", "Set the tag name")
	cmd.Flags().BoolVar(&flagDraft, "draft", false, "Set draft status")
	cmd.Flags().BoolVar(&flagPrerelease, "prerelease", false, "Set prerelease status")
	return cmd
}

func releaseDeleteCmd() *cobra.Command {
	var flagYes bool

	cmd := &cobra.Command{
		Use:   "delete <tag>",
		Short: "Delete a release",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := args[0]

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			if !flagYes {
				if err := confirm(fmt.Sprintf("Delete release %q in %s/%s? [y/N] ", tag, owner, repoName)); err != nil {
					return err
				}
			}

			if err := forge.Releases().Delete(cmd.Context(), owner, repoName, tag); err != nil {
				return notSupported(err, "releases")
			}

			_, _ = fmt.Fprintf(os.Stdout, "Deleted %s\n", tag)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagYes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func releaseUploadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upload <tag> <file>",
		Short: "Upload an asset to a release",
		Args:  cobra.ExactArgs(releaseAssetArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := args[0]
			filePath := args[1]

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			f, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("opening file: %w", err)
			}
			defer func() { _ = f.Close() }()

			asset, err := forge.Releases().UploadAsset(cmd.Context(), owner, repoName, tag, f)
			if err != nil {
				return notSupported(err, "releases")
			}

			p := printer()
			if p.Format == output.JSON {
				return p.PrintJSON(asset)
			}

			_, _ = fmt.Fprintf(os.Stdout, "%s (%d bytes)\n", asset.Name, asset.Size)
			if asset.DownloadURL != "" {
				_, _ = fmt.Fprintln(os.Stdout, asset.DownloadURL)
			}
			return nil
		},
	}
}

func releaseDownloadCmd() *cobra.Command {
	var flagOutput string

	cmd := &cobra.Command{
		Use:   "download <tag> <asset-name>",
		Short: "Download a release asset",
		Long:  "Downloads a release asset by first listing assets to find the matching name, then downloading by ID.",
		Args:  cobra.ExactArgs(releaseAssetArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag := args[0]
			assetName := args[1]

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			release, err := forge.Releases().Get(cmd.Context(), owner, repoName, tag)
			if err != nil {
				return notSupported(err, "releases")
			}

			var assetID int64
			for _, a := range release.Assets {
				if a.Name == assetName {
					assetID = a.ID
					break
				}
			}
			if assetID == 0 {
				return fmt.Errorf("asset %q not found in release %s", assetName, tag)
			}

			rc, err := forge.Releases().DownloadAsset(cmd.Context(), owner, repoName, assetID)
			if err != nil {
				return notSupported(err, "releases")
			}
			defer func() { _ = rc.Close() }()

			outPath := flagOutput
			if outPath == "" {
				outPath = assetName
			}

			out, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("creating output file: %w", err)
			}
			defer func() { _ = out.Close() }()

			if _, err := out.ReadFrom(rc); err != nil {
				return fmt.Errorf("writing file: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Downloaded %s\n", outPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagOutput, "output-file", "O", "", "Output file path (defaults to asset name)")
	return cmd
}

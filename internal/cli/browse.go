package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

var browseCmd = &cobra.Command{
	Use:   "browse [<number> | <path>]",
	Short: "Open in the browser",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		forge, owner, repoName, domain, err := resolve.Repo(flagRepo, flagForgeType)
		if err != nil {
			return err
		}

		repo, err := forge.Repos().Get(cmd.Context(), owner, repoName)
		if err != nil {
			return err
		}

		url := repo.HTMLURL
		if url == "" {
			url = fmt.Sprintf("https://%s/%s/%s", domain, owner, repoName)
		}

		if flagBrowseSettings {
			url += "/settings"
		} else if flagBrowseWiki {
			url += "/wiki"
		} else if flagBrowseActions {
			url += "/actions"
		} else if flagBrowseReleases {
			url += "/releases"
		} else if flagBrowseIssues {
			url += "/issues"
		} else if flagBrowsePulls {
			url += "/pulls"
		} else if len(args) > 0 {
			if n, err := strconv.Atoi(args[0]); err == nil {
				url += fmt.Sprintf("/issues/%d", n)
			} else {
				branch := flagBrowseBranch
				if branch == "" {
					branch = repo.DefaultBranch
				}
				url += fmt.Sprintf("/blob/%s/%s", branch, args[0])
			}
		}

		if flagNoBrowser {
			_, _ = fmt.Fprintln(os.Stdout, url)
			return nil
		}

		return openBrowser(url)
	},
}

var (
	flagBrowseBranch   string
	flagNoBrowser      bool
	flagBrowseSettings bool
	flagBrowseWiki     bool
	flagBrowseActions  bool
	flagBrowseReleases bool
	flagBrowseIssues   bool
	flagBrowsePulls    bool
)

func init() {
	rootCmd.AddCommand(browseCmd)
	browseCmd.Flags().StringVarP(&flagBrowseBranch, "branch", "b", "", "Branch for file URLs")
	browseCmd.Flags().BoolVarP(&flagNoBrowser, "no-browser", "n", false, "Print URL instead of opening")
	browseCmd.Flags().BoolVar(&flagBrowseSettings, "settings", false, "Open settings page")
	browseCmd.Flags().BoolVar(&flagBrowseWiki, "wiki", false, "Open wiki page")
	browseCmd.Flags().BoolVar(&flagBrowseActions, "actions", false, "Open actions page")
	browseCmd.Flags().BoolVar(&flagBrowseReleases, "releases", false, "Open releases page")
	browseCmd.Flags().BoolVar(&flagBrowseIssues, "issues", false, "Open issues page")
	browseCmd.Flags().BoolVar(&flagBrowsePulls, "pulls", false, "Open pull requests page")
}

func openBrowser(url string) error {
	argv := browserCmd(runtime.GOOS, url)
	return exec.Command(argv[0], argv[1:]...).Start()
}

// browserCmd returns the argv to open url in a browser. The BROWSER
// environment variable takes precedence over the platform default.
//
// On Windows we use rundll32 rather than cmd /c start, because cmd.exe
// re-parses its /c argument: a URL containing & (which can come from
// repo.HTMLURL or repo.DefaultBranch returned by a malicious forge) would
// be split into separate shell commands.
func browserCmd(goos, url string) []string {
	if exe := os.Getenv("BROWSER"); exe != "" {
		return []string{exe, url}
	}
	switch goos {
	case "linux":
		return []string{"xdg-open", url}
	case "windows":
		return []string{"rundll32", "url.dll,FileProtocolHandler", url}
	default:
		return []string{"open", url}
	}
}

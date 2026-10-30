package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/git-pkgs/forge/gitea"
	"github.com/git-pkgs/forge/internal/config"
	"github.com/git-pkgs/forge/internal/resolve"
)

func TestValidateBrowserURL(t *testing.T) {
	tests := []struct {
		name, url, domain string
		valid             bool
	}{
		{"https", "https://github.com/owner/repo", "github.com", true},
		{"http", "http://forge.local:3000/owner/repo", "forge.local:3000", true},
		{"case insensitive", "HTTPS://FORGE.EXAMPLE/owner/repo", "forge.example", true},
		{"ipv6", "http://[::1]:3000/owner/repo", "[::1]:3000", true},
		{"query and fragment", "https://forge.example/owner/repo?q=one&sort=asc#L1", "forge.example", true},
		{"escaped path", "https://forge.example/owner/repo/blob/feature%2Fone/file%20name", "forge.example", true},
		{"empty", "", "forge.example", false},
		{"local path", "/tmp/repo.html", "forge.example", false},
		{"option", "-aCalculator", "forge.example", false},
		{"relative", "owner/repo", "forge.example", false},
		{"scheme relative", "//forge.example/owner/repo", "forge.example", false},
		{"file", "file:///tmp/repo.html", "forge.example", false},
		{"custom scheme", "forge-probe://forge.example/action", "forge.example", false},
		{"javascript", "javascript:alert(1)", "forge.example", false},
		{"opaque", "https:forge.example/owner/repo", "forge.example", false},
		{"missing host", "https:///owner/repo", "forge.example", false},
		{"empty hostname", "https://:443/owner/repo", ":443", false},
		{"other host", "https://other.example/owner/repo", "forge.example", false},
		{"host suffix", "https://forge.example.other.example/owner/repo", "forge.example", false},
		{"other port", "http://forge.local:3001/owner/repo", "forge.local:3000", false},
		{"userinfo", "https://user:password@forge.example/owner/repo", "forge.example", false},
		{"empty userinfo", "https://@forge.example/owner/repo", "forge.example", false},
		{"host in userinfo", "https://forge.example@other.example/owner/repo", "forge.example", false},
		{"backslash", `https://other.example\@forge.example/owner/repo`, "forge.example", false},
		{"control character", "https://forge.example/owner/repo\n", "forge.example", false},
		{"invalid escape", "https://forge.example/%zz", "forge.example", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBrowserURL(tt.url, tt.domain)
			if (err == nil) != tt.valid {
				t.Fatalf("validateBrowserURL(%q, %q) = %v, valid = %v", tt.url, tt.domain, err, tt.valid)
			}
		})
	}
}

func TestBrowserCommandsRejectUnsafeURLs(t *testing.T) {
	commands := [][]string{
		{"browse"},
		{"browse", "7"},
		{"browse", "README.md"},
		{"browse", "--settings"},
		{"browse", "--wiki"},
		{"browse", "--actions"},
		{"browse", "--releases"},
		{"browse", "--issues"},
		{"browse", "--pulls"},
		{"issue", "view", "7", "--web"},
		{"issue", "list", "--web"},
		{"pr", "view", "7", "--web"},
		{"pr", "list", "--web"},
		{"label", "list", "--web"},
	}
	for _, args := range commands {
		for _, target := range []string{"file:///tmp/repo.html", "https://other.example/owner/repo"} {
			t.Run(strings.Join(args, " ")+"/"+target, func(t *testing.T) {
				resetCmd(rootCmd)
				t.Cleanup(func() {
					resetCmd(rootCmd)
					rootCmd.SetArgs(nil)
				})
				t.Setenv("XDG_CONFIG_HOME", t.TempDir())
				t.Setenv("BROWSER", filepath.Join(t.TempDir(), "missing-browser"))
				config.ResetCache()
				t.Cleanup(config.ResetCache)

				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					switch r.URL.Path {
					case "/api/v1/repos/owner/repo":
						repoURL := target
						if slices.Contains(args, "7") {
							repoURL = "https://forge.example/owner/repo"
						}
						_ = json.NewEncoder(w).Encode(map[string]any{
							"name": "repo", "full_name": "owner/repo", "owner": map[string]string{"login": "owner"},
							"html_url": repoURL, "default_branch": "main",
						})
					case "/api/v1/repos/owner/repo/topics":
						_, _ = io.WriteString(w, `{"topics":[]}`)
					case "/api/v1/repos/owner/repo/issues/7", "/api/v1/repos/owner/repo/pulls/7":
						_ = json.NewEncoder(w).Encode(map[string]any{
							"number": 7, "title": "Example", "state": "closed", "html_url": target,
						})
					default:
						t.Errorf("unexpected API request: %s", r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				defer server.Close()
				resolve.SetTestForge(gitea.New(server.URL, "", server.Client()), "owner", "repo", "forge.example")
				t.Cleanup(resolve.ResetTestForge)
				rootCmd.SetArgs(args)
				err := rootCmd.Execute()
				if err == nil || !strings.Contains(err.Error(), "invalid browser URL:") {
					t.Fatalf("expected URL rejection before launching browser, got %v", err)
				}
			})
		}
	}
}

func TestBrowserCmd(t *testing.T) {
	const url = "https://github.com/owner/repo"

	tests := []struct {
		name    string
		goos    string
		browser string
		want    []string
	}{
		{
			name: "darwin default",
			goos: "darwin",
			want: []string{"open", url},
		},
		{
			name: "linux default",
			goos: "linux",
			want: []string{"xdg-open", url},
		},
		{
			name: "windows default uses rundll32",
			goos: "windows",
			want: []string{"rundll32", "url.dll,FileProtocolHandler", url},
		},
		{
			name:    "BROWSER overrides darwin",
			goos:    "darwin",
			browser: "firefox",
			want:    []string{"firefox", url},
		},
		{
			name:    "BROWSER overrides linux",
			goos:    "linux",
			browser: "/usr/local/bin/chromium",
			want:    []string{"/usr/local/bin/chromium", url},
		},
		{
			name:    "BROWSER overrides windows",
			goos:    "windows",
			browser: "firefox.exe",
			want:    []string{"firefox.exe", url},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.browser != "" {
				t.Setenv("BROWSER", tt.browser)
			} else {
				t.Setenv("BROWSER", "")
			}

			got := browserCmd(tt.goos, url)
			if !slices.Equal(got, tt.want) {
				t.Errorf("browserCmd(%q, %q) = %v, want %v", tt.goos, url, got, tt.want)
			}
		})
	}
}

func TestBrowserCmdWindowsDoesNotInvokeCmdShell(t *testing.T) {
	// cmd.exe re-parses its /c argument, so a URL containing & or | would
	// be split into separate commands. rundll32 receives the URL as a single
	// argv element with no shell interpretation.
	t.Setenv("BROWSER", "")

	hostile := "https://evil.com/&calc.exe"
	got := browserCmd("windows", hostile)

	if got[0] == "cmd" {
		t.Fatalf("windows browser command must not use cmd.exe: %v", got)
	}
	if got[len(got)-1] != hostile {
		t.Errorf("url should be the final argv element unchanged, got %v", got)
	}
}

func TestBrowserCmdURLIsSingleArgv(t *testing.T) {
	// The URL must always be exactly one argv element so shell metacharacters
	// in branch names or paths from the API cannot be split.
	t.Setenv("BROWSER", "")

	url := "https://example.com/blob/feat;rm -rf/file"
	for _, goos := range []string{"darwin", "linux", "windows"} {
		got := browserCmd(goos, url)
		found := 0
		for _, a := range got {
			if a == url {
				found++
			}
		}
		if found != 1 {
			t.Errorf("%s: url should appear exactly once as a whole argv element, got %v", goos, got)
		}
	}
}

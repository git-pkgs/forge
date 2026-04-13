package cli

import (
	"slices"
	"testing"
)

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

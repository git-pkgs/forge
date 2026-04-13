package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func TestRepoCmd(t *testing.T) {
	// Test that the repo command is registered and has the expected subcommands
	cmd := repoCmd
	if cmd.Use != "repo" {
		t.Errorf("expected Use=repo, got %s", cmd.Use)
	}

	subcommands := cmd.Commands()
	want := map[string]bool{
		"view":   false,
		"list":   false,
		"create": false,
		"edit":   false,
		"delete": false,
		"fork":   false,
		"clone":  false,
		"search": false,
	}

	for _, sub := range subcommands {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestRootCmd(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("forge")) {
		t.Error("help output should mention forge")
	}
}

func TestRepoCreateMutuallyExclusiveVisibility(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"repo", "create", "test-repo", "--private", "--public"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for conflicting visibility flags")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' in error, got: %s", err)
	}
}

func TestRepoEditMutuallyExclusiveVisibility(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"repo", "edit", "owner/repo", "--private", "--public"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for conflicting visibility flags")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' in error, got: %s", err)
	}
}

func TestDomainFromFlags(t *testing.T) {
	tests := []struct {
		forgeType string
		want      string
	}{
		{"", "github.com"},
		{"github", "github.com"},
		{"gitlab", "gitlab.com"},
		{"gitea", "codeberg.org"},
		{"forgejo", "codeberg.org"},
		{"bitbucket", "bitbucket.org"},
	}

	for _, tt := range tests {
		flagForgeType = tt.forgeType
		got := domainFromFlags()
		if got != tt.want {
			t.Errorf("forgeType=%q: want %q, got %q", tt.forgeType, tt.want, got)
		}
	}
	flagForgeType = "" // reset
}

func TestGitCloneArgs(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want []string
	}{
		{
			name: "https url",
			url:  "https://github.com/owner/repo.git",
			want: []string{"clone", "--", "https://github.com/owner/repo.git"},
		},
		{
			name: "ssh url",
			url:  "git@github.com:owner/repo.git",
			want: []string{"clone", "--", "git@github.com:owner/repo.git"},
		},
		{
			name: "url that looks like a git option",
			url:  "--upload-pack=evil",
			want: []string{"clone", "--", "--upload-pack=evil"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gitCloneArgs(tt.url)
			if !slices.Equal(got, tt.want) {
				t.Errorf("gitCloneArgs(%q) = %v, want %v", tt.url, got, tt.want)
			}

			// The url must appear after the -- separator so git cannot
			// parse a server-supplied CloneURL as an option.
			sep := slices.Index(got, "--")
			urlIdx := slices.Index(got, tt.url)
			if sep == -1 {
				t.Fatal("expected -- separator in argv")
			}
			if urlIdx <= sep {
				t.Errorf("url at index %d is not after -- at index %d", urlIdx, sep)
			}
		})
	}
}

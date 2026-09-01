package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/config"
	"github.com/git-pkgs/forge/internal/resolve"
)

func TestNotSupportedWrapsError(t *testing.T) {
	err := notSupported(forges.ErrNotSupported, "CI pipelines")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	want := "CI pipelines is not supported by this forge"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestNotSupportedPassesThrough(t *testing.T) {
	original := errors.New("connection refused")
	err := notSupported(original, "CI pipelines")
	if err != original {
		t.Errorf("expected original error %q, got %q", original, err)
	}
}

func TestRemoteFlagHelp(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("remote")
	if flag == nil {
		t.Fatal("remote flag not found")
	}

	want := "Git remote to use when --repo (-R) is not specified (default from config, otherwise origin)"
	if flag.Usage != want {
		t.Errorf("remote flag help = %q, want %q", flag.Usage, want)
	}
}

func TestRemotePrecedence(t *testing.T) {
	tests := []struct {
		name         string
		configRemote string
		flagRemote   string
		want         string
	}{
		{name: "origin fallback", want: "origin"},
		{name: "config remote", configRemote: "devrepo", want: "devrepo"},
		{name: "flag overrides config", configRemote: "devrepo", flagRemote: "fork", want: "fork"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetCmd(rootCmd)
			config.ResetCache()
			t.Cleanup(config.ResetCache)
			resolve.SetRemote("origin")
			t.Cleanup(func() { resolve.SetRemote("origin") })

			dir := t.TempDir()
			t.Chdir(dir)
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
			if tt.configRemote != "" {
				configDir := filepath.Join(dir, "config", "forge")
				if err := os.MkdirAll(configDir, 0o700); err != nil {
					t.Fatal(err)
				}
				contents := []byte("[default]\nremote = " + tt.configRemote + "\n")
				if err := os.WriteFile(filepath.Join(configDir, "config"), contents, 0o600); err != nil {
					t.Fatal(err)
				}
			}

			if tt.flagRemote != "" {
				if err := rootCmd.PersistentFlags().Set("remote", tt.flagRemote); err != nil {
					t.Fatal(err)
				}
			}
			rootCmd.PersistentPreRun(rootCmd, nil)

			if got := resolve.RemoteName(); got != tt.want {
				t.Fatalf("remote = %q, want %q", got, tt.want)
			}
		})
	}
}

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
		wantRepo     string
	}{
		{name: "origin fallback", wantRepo: "origin-owner/origin-repo"},
		{name: "config remote", configRemote: "devrepo", wantRepo: "config-owner/config-repo"},
		{name: "flag overrides config", configRemote: "devrepo", flagRemote: "fork", wantRepo: "flag-owner/flag-repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetCmd(rootCmd)
			t.Cleanup(func() {
				resetCmd(rootCmd)
				rootCmd.SetArgs(nil)
			})
			resolve.ResetTestForge()
			t.Cleanup(resolve.ResetTestForge)
			config.ResetCache()
			t.Cleanup(config.ResetCache)
			resolve.SetRemote("origin")
			t.Cleanup(func() { resolve.SetRemote("origin") })
			t.Setenv("FORGE_HOST", "")
			t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
			t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

			requestedRepo := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := strings.TrimPrefix(r.URL.Path, "/api/v1/repos/")
				if strings.HasSuffix(path, "/topics") {
					_ = json.NewEncoder(w).Encode(map[string]any{"topics": []string{}})
					return
				}

				requestedRepo <- path
				parts := strings.SplitN(path, "/", 2)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"full_name": path,
					"name":      parts[1],
					"owner":     map[string]any{"login": parts[0]},
				})
			}))
			defer server.Close()

			repoDir := setupTestRepo(t, server.URL+"/origin-owner/origin-repo.git")
			mustGit(t, repoDir, "remote", "add", "devrepo", server.URL+"/config-owner/config-repo.git")
			mustGit(t, repoDir, "remote", "add", "fork", server.URL+"/flag-owner/flag-repo.git")
			t.Chdir(repoDir)

			serverURL, err := url.Parse(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			configHome := t.TempDir()
			configDir := filepath.Join(configHome, "forge")
			if err := os.MkdirAll(configDir, 0o700); err != nil {
				t.Fatal(err)
			}
			configContents := "[default]\n"
			if tt.configRemote != "" {
				configContents += "remote = " + tt.configRemote + "\n"
			}
			configContents += fmt.Sprintf("\n[%s]\ntype = gitea\nscheme = http\n", serverURL.Host)
			if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(configContents), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("XDG_CONFIG_HOME", configHome)

			args := []string{"--output", "json"}
			if tt.flagRemote != "" {
				args = append(args, "--remote", tt.flagRemote)
			}
			rootCmd.SetArgs(append(args, "repo", "view"))
			stdout, err := captureStdout(t, rootCmd.Execute)
			if err != nil {
				t.Fatalf("root command: %v", err)
			}

			if got := <-requestedRepo; got != tt.wantRepo {
				t.Fatalf("requested repo = %q, want %q", got, tt.wantRepo)
			}
			if !strings.Contains(stdout, tt.wantRepo) {
				t.Fatalf("output does not contain selected repo %q: %s", tt.wantRepo, stdout)
			}
		})
	}
}

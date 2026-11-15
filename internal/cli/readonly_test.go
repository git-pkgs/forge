package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/config"
	"github.com/git-pkgs/forge/internal/resolve"
)

// runReadOnlyCLI runs the root command against a fake Gitea server, which is
// probed when detect is set and configured as Gitea otherwise, and
// returns the writes the server received and the command's error.
func runReadOnlyCLI(t *testing.T, env string, detect bool, args ...string) ([]string, error) {
	t.Helper()
	resetCmd(rootCmd)
	t.Cleanup(func() {
		resetCmd(rootCmd)
		rootCmd.SetArgs(nil)
		resolve.SetReadOnly(false)
	})
	resolve.ResetTestForge()
	config.ResetCache()
	t.Cleanup(config.ResetCache)
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("FORGE_HOST", "")
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("FORGE_READ_ONLY", env)

	var (
		mu     sync.Mutex
		writes []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/":
			w.Header().Set("X-Gitea-Version", "1.26.0")
		case r.URL.Path == "/api/v1/version":
			_, _ = fmt.Fprint(w, `{"version":"1.26.0"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/owner/repo/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "name": "bug", "color": "ff0000"}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/owner/repo/labels":
			mu.Lock()
			writes = append(writes, r.Method+" "+r.URL.Path)
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 2, "name": "docs", "color": "0000ff"})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	configDir := filepath.Join(configHome, "forge")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	section := "[" + host + "]\nscheme = http\n"
	if !detect {
		section += "type = gitea\n"
	}
	if err := os.WriteFile(filepath.Join(configDir, "config"), []byte(section), 0o600); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs(append([]string{"-R", host + "/owner/repo"}, args...))
	_, err := captureStdout(t, rootCmd.Execute)

	mu.Lock()
	defer mu.Unlock()
	return writes, err
}

func TestReadOnlyRefusesWrites(t *testing.T) {
	tests := []struct {
		name   string
		env    string
		detect bool
		args   []string
	}{
		{name: "flag", args: []string{"--read-only"}},
		{name: "env true", env: "true"},
		{name: "env 1", env: "1"},
		{name: "flag overrides env false", env: "false", args: []string{"--read-only"}},
		{name: "detected forge", detect: true, args: []string{"--read-only"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := runReadOnlyCLI(t, tt.env, tt.detect, append(tt.args, "label", "list")...); err != nil {
				t.Fatalf("label list: %v", err)
			}

			writes, err := runReadOnlyCLI(t, tt.env, tt.detect, append(tt.args, "label", "create", "docs", "--color", "0000ff")...)
			if !errors.Is(err, forges.ErrReadOnly) {
				t.Fatalf("label create: err = %v, want ErrReadOnly", err)
			}
			if len(writes) != 0 {
				t.Errorf("server received writes: %v", writes)
			}
		})
	}
}

func TestReadOnlyRefusesAPIWrites(t *testing.T) {
	writes, err := runReadOnlyCLI(t, "", false, "--read-only", "api", "-X", "POST", "repos/{owner}/{repo}/labels")
	if !errors.Is(err, forges.ErrReadOnly) {
		t.Fatalf("api -X POST: err = %v, want ErrReadOnly", err)
	}
	if len(writes) != 0 {
		t.Errorf("server received writes: %v", writes)
	}
}

func TestReadOnlyDisabled(t *testing.T) {
	tests := []struct {
		name   string
		env    string
		detect bool
		args   []string
	}{
		{name: "default"},
		{name: "env false", env: "false"},
		{name: "flag false overrides env", env: "true", args: []string{"--read-only=false"}},
		{name: "detected forge", detect: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writes, err := runReadOnlyCLI(t, tt.env, tt.detect, append(tt.args, "label", "create", "docs", "--color", "0000ff")...)
			if err != nil {
				t.Fatalf("label create: %v", err)
			}
			if len(writes) != 1 {
				t.Errorf("server received writes %v, want one POST", writes)
			}
		})
	}
}

func TestReadOnlyInvalidEnv(t *testing.T) {
	_, err := runReadOnlyCLI(t, "maybe", false, "label", "list")
	if err == nil || !strings.Contains(err.Error(), `invalid FORGE_READ_ONLY value "maybe"`) {
		t.Fatalf("err = %v, want invalid FORGE_READ_ONLY error", err)
	}
}

func TestPRCreatePushRefusedInReadOnlyMode(t *testing.T) {
	t.Chdir(t.TempDir())
	prs := &mockPRService{qualifiedHeads: true}
	resolve.SetTestForge(&mockForge{prService: prs}, "owner", "repo", "example.com")
	t.Cleanup(resolve.ResetTestForge)
	resolve.SetReadOnly(true)
	t.Cleanup(func() { resolve.SetReadOnly(false) })

	cmd := prCreateCmd()
	cmd.SetArgs([]string{"--title", "Test PR", "--head", "feature", "--push"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--push cannot be used in read-only mode") {
		t.Fatalf("pr create --push error = %v, want read-only error", err)
	}
	if prs.createCalls != 0 {
		t.Errorf("Create calls = %d, want 0", prs.createCalls)
	}
}

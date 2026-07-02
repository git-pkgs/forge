package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-pkgs/forge/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// resetCmd resets flag values and Changed state across the command tree so
// tests using the shared rootCmd don't leak flag state into each other.
func resetCmd(cmd *cobra.Command) {
	reset := func(f *pflag.Flag) {
		f.Changed = false
		_ = f.Value.Set(f.DefValue)
	}
	cmd.Flags().VisitAll(reset)
	cmd.PersistentFlags().VisitAll(reset)
	for _, sub := range cmd.Commands() {
		resetCmd(sub)
	}
}

func TestAuthCmd(t *testing.T) {
	cmd := authCmd
	if cmd.Use != "auth" {
		t.Errorf("expected Use=auth, got %s", cmd.Use)
	}

	subcommands := cmd.Commands()
	want := map[string]bool{
		"login":  false,
		"status": false,
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

func TestAuthLoginRequiresDomainNonInteractive(t *testing.T) {
	resetCmd(rootCmd)
	// Replace stdin with a pipe so term.IsTerminal returns false
	origStdin := os.Stdin
	r, w, _ := os.Pipe()
	_ = w.Close()
	os.Stdin = r
	defer func() { os.Stdin = origStdin; _ = r.Close() }()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"auth", "login", "--domain", ""})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when domain not provided non-interactively")
	}
	if !strings.Contains(err.Error(), "--domain is required") {
		t.Errorf("expected domain required error, got: %v", err)
	}
}

func TestAuthLoginNonInteractive(t *testing.T) {
	resetCmd(rootCmd)
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	config.ResetCache()
	defer config.ResetCache()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{
		"auth", "login",
		"--domain", "gitea.example.com",
		"--token", "test_token_123",
		"--type", "gitea",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the config was written
	data, err := os.ReadFile(filepath.Join(dir, "forge", "config"))
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "[gitea.example.com]") {
		t.Error("expected domain section in config file")
	}
	if !strings.Contains(content, "token = test_token_123") {
		t.Error("expected token in config file")
	}
	if !strings.Contains(content, "type = gitea") {
		t.Error("expected type in config file")
	}
}

func TestAuthLoginTokenCmd(t *testing.T) {
	resetCmd(rootCmd)
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	config.ResetCache()
	defer config.ResetCache()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{
		"auth", "login",
		"--domain", "github.com",
		"--token-cmd", "rbw get github-token",
	})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "forge", "config"))
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "token-cmd = rbw get github-token") {
		t.Errorf("expected token command in config, got:\n%s", content)
	}
}

func TestAuthLoginTokenAndTokenCmdMutuallyExclusive(t *testing.T) {
	resetCmd(rootCmd)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{
		"auth", "login",
		"--domain", "github.com",
		"--token", "ghp_abc",
		"--token-cmd", "rbw get github-token",
	})

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error when both --token and --token-cmd are set")
	}
}

func TestAuthStatus(t *testing.T) {
	resetCmd(rootCmd)
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	config.ResetCache()
	defer config.ResetCache()

	cfgDir := filepath.Join(dir, "forge")
	_ = os.MkdirAll(cfgDir, 0700)
	_ = os.WriteFile(filepath.Join(cfgDir, "config"), []byte(`[gitea.example.com]
type = gitea
token = some_token
`), 0600)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"auth", "status"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "gitea.example.com") {
		t.Errorf("expected domain in output, got: %s", out)
	}
	if !strings.Contains(out, "token from config") {
		t.Errorf("expected token source in output, got: %s", out)
	}
}

func TestAuthStatusWithTokenCmd(t *testing.T) {
	resetCmd(rootCmd)
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	config.ResetCache()
	defer config.ResetCache()

	cfgDir := filepath.Join(dir, "forge")
	_ = os.MkdirAll(cfgDir, 0700)
	_ = os.WriteFile(filepath.Join(cfgDir, "config"), []byte(`[gitlab.example.com]
type = gitlab
token-cmd = echo secret
`), 0600)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"auth", "status"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "cmd: echo secret") {
		t.Errorf("expected command source in output, got: %s", out)
	}
}

func TestReadOneByte(t *testing.T) {
	b, err := readOneByte(bytes.NewReader([]byte{'x'}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b != 'x' {
		t.Errorf("expected 'x', got %q", b)
	}
}

func TestReadCommandInteractive(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin; _ = r.Close() }()

	_, _ = w.WriteString("rbw get github-token\n")
	_ = w.Close()

	result, err := readCommandInteractive("github.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "rbw get github-token" {
		t.Errorf("expected %q, got %q", "rbw get github-token", result)
	}
}

func TestReadCommandInteractiveEmpty(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin; _ = r.Close() }()

	_, _ = w.WriteString("\n")
	_ = w.Close()

	_, err = readCommandInteractive("github.com")
	if err == nil {
		t.Fatal("expected error for empty command")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("expected empty command error, got: %v", err)
	}
}

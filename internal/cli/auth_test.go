package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-pkgs/forges/internal/config"
)

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
	// Replace stdin with a pipe so term.IsTerminal returns false
	origStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = origStdin; r.Close() }()

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

func TestAuthStatus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	config.ResetCache()
	defer config.ResetCache()

	// Write a config with a domain
	cfgDir := filepath.Join(dir, "forge")
	os.MkdirAll(cfgDir, 0700)
	os.WriteFile(filepath.Join(cfgDir, "config"), []byte(`[gitea.example.com]
type = gitea
token = some_token
`), 0600)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"auth", "status"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

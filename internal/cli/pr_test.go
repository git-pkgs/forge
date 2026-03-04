package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPRCmd(t *testing.T) {
	cmd := prCmd
	if cmd.Use != "pr" {
		t.Errorf("expected Use=pr, got %s", cmd.Use)
	}

	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "mr" {
		t.Errorf("expected alias mr, got %v", cmd.Aliases)
	}

	subcommands := cmd.Commands()
	want := map[string]bool{
		"view":    false,
		"list":    false,
		"create":  false,
		"close":   false,
		"reopen":  false,
		"edit":    false,
		"merge":   false,
		"diff":    false,
		"comment": false,
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

func TestPRCmdAlias(t *testing.T) {
	// Verify the mr alias is registered on the root command
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "pr" {
			for _, alias := range cmd.Aliases {
				if alias == "mr" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected mr alias on pr command")
	}
}

func TestPRViewInvalidNumber(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"pr", "view", "notanumber"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-numeric PR number")
	}
	if !strings.Contains(err.Error(), "invalid PR number") {
		t.Errorf("expected 'invalid PR number' in error, got: %s", err)
	}
}

func TestPRViewRequiresArg(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"pr", "view"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing argument")
	}
}

func TestPRCreateRequiresTitleAndHead(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"pr", "create"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing title")
	}
	if !strings.Contains(err.Error(), "--title is required") {
		t.Errorf("expected '--title is required' in error, got: %s", err)
	}
}

func TestPRCreateRequiresHead(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"pr", "create", "--title", "test"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing head")
	}
	if !strings.Contains(err.Error(), "--head is required") {
		t.Errorf("expected '--head is required' in error, got: %s", err)
	}
}

func TestPRMergeInvalidNumber(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"pr", "merge", "abc"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid PR number") {
		t.Errorf("expected 'invalid PR number' in error, got: %s", err)
	}
}

func TestPRDiffRequiresArg(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"pr", "diff"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing argument")
	}
}

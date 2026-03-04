package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestIssueCmd(t *testing.T) {
	cmd := issueCmd
	if cmd.Use != "issue" {
		t.Errorf("expected Use=issue, got %s", cmd.Use)
	}

	subcommands := cmd.Commands()
	want := map[string]bool{
		"view":    false,
		"list":    false,
		"create":  false,
		"close":   false,
		"reopen":  false,
		"edit":    false,
		"delete":  false,
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

func TestIssueViewInvalidNumber(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"issue", "view", "notanumber"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-numeric issue number")
	}
	if !strings.Contains(err.Error(), "invalid issue number") {
		t.Errorf("expected 'invalid issue number' in error, got: %s", err)
	}
}

func TestIssueViewRequiresArg(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"issue", "view"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing argument")
	}
}

func TestIssueCreateRequiresTitle(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"issue", "create", "--body", "test body"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing title")
	}
	if !strings.Contains(err.Error(), "--title is required") {
		t.Errorf("expected '--title is required' in error, got: %s", err)
	}
}

func TestIssueCloseInvalidNumber(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"issue", "close", "xyz"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid issue number") {
		t.Errorf("expected 'invalid issue number' in error, got: %s", err)
	}
}

func TestIssueDeleteInvalidNumber(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"issue", "delete", "xyz"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid issue number") {
		t.Errorf("expected 'invalid issue number' in error, got: %s", err)
	}
}

package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCommentRequiresNumber(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"comment"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing number argument")
	}
}

func TestCommentInvalidNumber(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"comment", "abc", "--body", "test"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid number")
	}
	if !strings.Contains(err.Error(), "invalid number") {
		t.Errorf("expected 'invalid number' in error, got: %s", err)
	}
}

func TestCommentRequiresBody(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"comment", "1"})
	flagCommentBody = ""

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing body")
	}
	if !strings.Contains(err.Error(), "--body is required") {
		t.Errorf("expected '--body is required' in error, got: %s", err)
	}
}

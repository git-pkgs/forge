package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestReleaseEditDraftFlag(t *testing.T) {
	cmd := releaseEditCmd()

	// The --draft flag should be a bool, not a string.
	f := cmd.Flags().Lookup("draft")
	if f == nil {
		t.Fatal("expected --draft flag to exist")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("expected --draft to be bool, got %s", f.Value.Type())
	}

	f = cmd.Flags().Lookup("prerelease")
	if f == nil {
		t.Fatal("expected --prerelease flag to exist")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("expected --prerelease to be bool, got %s", f.Value.Type())
	}
}

func TestReleaseCreateDraftFlag(t *testing.T) {
	cmd := releaseCreateCmd()

	f := cmd.Flags().Lookup("draft")
	if f == nil {
		t.Fatal("expected --draft flag to exist")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("expected --draft to be bool, got %s", f.Value.Type())
	}
}

// Ensure the cobra Command type is used (avoid import cycle).
var _ *cobra.Command

func TestReleaseSubcommands(t *testing.T) {
	subs := releaseCmd.Commands()
	want := map[string]bool{
		"list":     false,
		"view":     false,
		"create":   false,
		"edit":     false,
		"delete":   false,
		"upload":   false,
		"download": false,
	}

	for _, cmd := range subs {
		if _, ok := want[cmd.Name()]; ok {
			want[cmd.Name()] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

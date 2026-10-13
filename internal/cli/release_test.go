package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

const flagTypeBool = "bool"

func TestReleaseEditDraftFlag(t *testing.T) {
	cmd := releaseEditCmd()

	// The --draft flag should be a bool, not a string.
	f := cmd.Flags().Lookup("draft")
	if f == nil {
		t.Fatal("expected --draft flag to exist")
	}
	if f.Value.Type() != flagTypeBool {
		t.Errorf("expected --draft to be bool, got %s", f.Value.Type())
	}

	f = cmd.Flags().Lookup("prerelease")
	if f == nil {
		t.Fatal("expected --prerelease flag to exist")
	}
	if f.Value.Type() != flagTypeBool {
		t.Errorf("expected --prerelease to be bool, got %s", f.Value.Type())
	}
}

func TestReleaseCreateDraftFlag(t *testing.T) {
	cmd := releaseCreateCmd()

	f := cmd.Flags().Lookup("draft")
	if f == nil {
		t.Fatal("expected --draft flag to exist")
	}
	if f.Value.Type() != flagTypeBool {
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

func TestReleaseUploadTagFlag(t *testing.T) {
	cmd := releaseUploadCmd()

	f := cmd.Flags().Lookup("tag")
	if f == nil {
		t.Fatal("expected --tag flag to exist")
	}
	if f.Value.Type() != "string" {
		t.Errorf("expected --tag to be string, got %s", f.Value.Type())
	}
}

func TestUploadTagAndFile(t *testing.T) {
	tests := []struct {
		name     string
		flagTag  string
		args     []string
		wantTag  string
		wantFile string
		wantErr  bool
	}{
		{name: "positional tag and file", args: []string{"v1.0.0", "app.zip"}, wantTag: "v1.0.0", wantFile: "app.zip"},
		{name: "tag flag and file", flagTag: "v1.0.0", args: []string{"app.zip"}, wantTag: "v1.0.0", wantFile: "app.zip"},
		{name: "tag flag with positional tag too", flagTag: "v1.0.0", args: []string{"v1.0.0", "app.zip"}, wantErr: true},
		{name: "no tag at all", args: []string{"app.zip"}, wantErr: true},
		{name: "no arguments", wantErr: true},
		{name: "tag flag without a file", flagTag: "v1.0.0", wantErr: true},
		{name: "too many positional arguments", args: []string{"v1.0.0", "a.zip", "b.zip"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag, file, err := uploadTagAndFile(tt.flagTag, tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("uploadTagAndFile(%q, %v) error = %v, wantErr %v", tt.flagTag, tt.args, err, tt.wantErr)
			}
			if tag != tt.wantTag || file != tt.wantFile {
				t.Errorf("uploadTagAndFile(%q, %v) = (%q, %q), want (%q, %q)",
					tt.flagTag, tt.args, tag, file, tt.wantTag, tt.wantFile)
			}
		})
	}
}

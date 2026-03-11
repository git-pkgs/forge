package cli

import (
	"strings"
	"testing"
)

func TestReactionCmdStructure(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "issue reactions requires number",
			args: []string{"issue", "reactions"},
			want: "accepts 1 arg",
		},
		{
			name: "issue react requires number",
			args: []string{"issue", "react"},
			want: "accepts 1 arg",
		},
		{
			name: "pr reactions requires number",
			args: []string{"pr", "reactions"},
			want: "accepts 1 arg",
		},
		{
			name: "pr react requires number",
			args: []string{"pr", "react"},
			want: "accepts 1 arg",
		},
		{
			name: "issue reactions requires comment flag",
			args: []string{"issue", "reactions", "1"},
			want: "--comment is required",
		},
		{
			name: "issue react requires comment flag",
			args: []string{"issue", "react", "1"},
			want: "--comment is required",
		},
		{
			name: "issue react requires reaction flag",
			args: []string{"issue", "react", "1", "--comment", "42"},
			want: "--reaction is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}

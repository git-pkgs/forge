package cli

import (
	"strings"
	"testing"
)

func TestReviewCmdStructure(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "review list requires number",
			args: []string{"pr", "review", "list"},
			want: "accepts 1 arg",
		},
		{
			name: "review approve requires number",
			args: []string{"pr", "review", "approve"},
			want: "accepts 1 arg",
		},
		{
			name: "review reject requires number",
			args: []string{"pr", "review", "reject"},
			want: "accepts 1 arg",
		},
		{
			name: "reviewer request requires number and users",
			args: []string{"pr", "reviewer", "request"},
			want: "requires at least 2 arg",
		},
		{
			name: "reviewer request requires users",
			args: []string{"pr", "reviewer", "request", "1"},
			want: "requires at least 2 arg",
		},
		{
			name: "reviewer remove requires number and users",
			args: []string{"pr", "reviewer", "remove"},
			want: "requires at least 2 arg",
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

func TestReviewRejectRequiresBody(t *testing.T) {
	rootCmd.SetArgs([]string{"pr", "review", "reject", "1"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--body is required") {
		t.Errorf("expected error about --body, got %q", err.Error())
	}
}

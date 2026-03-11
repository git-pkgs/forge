package cli

import (
	"testing"
)

func TestRateLimitCmdRejectsArgs(t *testing.T) {
	rootCmd.SetArgs([]string{"rate-limit", "extra"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for extra args")
	}
}

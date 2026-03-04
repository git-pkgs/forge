package cli

import (
	"testing"
)

func TestMilestoneCmd(t *testing.T) {
	cmd := milestoneCmd
	if cmd.Use != "milestone" {
		t.Errorf("expected Use=milestone, got %s", cmd.Use)
	}

	subcommands := cmd.Commands()
	want := map[string]bool{
		"list":   false,
		"view":   false,
		"create": false,
		"edit":   false,
		"close":  false,
		"reopen": false,
		"delete": false,
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

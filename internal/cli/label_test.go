package cli

import (
	"testing"
)

func TestLabelCmd(t *testing.T) {
	cmd := labelCmd
	if cmd.Use != "label" {
		t.Errorf("expected Use=label, got %s", cmd.Use)
	}

	subcommands := cmd.Commands()
	want := map[string]bool{
		"list":   false,
		"create": false,
		"edit":   false,
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

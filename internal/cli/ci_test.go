package cli

import "testing"

func TestCISubcommands(t *testing.T) {
	subs := ciCmd.Commands()
	want := map[string]bool{
		"list":   false,
		"view":   false,
		"run":    false,
		"cancel": false,
		"retry":  false,
		"log":    false,
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

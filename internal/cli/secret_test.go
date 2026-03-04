package cli

import "testing"

func TestSecretSubcommands(t *testing.T) {
	subs := secretCmd.Commands()
	want := map[string]bool{
		"list":   false,
		"set":    false,
		"delete": false,
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

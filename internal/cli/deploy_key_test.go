package cli

import "testing"

func TestDeployKeySubcommands(t *testing.T) {
	subs := deployKeyCmd.Commands()
	want := map[string]bool{
		"list":   false,
		"add":    false,
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

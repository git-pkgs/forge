package cli

import "testing"

func TestBranchSubcommands(t *testing.T) {
	subs := branchCmd.Commands()
	want := map[string]bool{
		"list":   false,
		"create": false,
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

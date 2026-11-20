package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/resolve"
)

type mockDeployKeyService struct {
	createErr error
}

func (m *mockDeployKeyService) List(context.Context, string, string, forges.ListDeployKeyOpts) ([]forges.DeployKey, error) {
	return nil, nil
}

func (m *mockDeployKeyService) Get(context.Context, string, string, int64) (*forges.DeployKey, error) {
	return nil, nil
}

func (m *mockDeployKeyService) Create(context.Context, string, string, forges.CreateDeployKeyOpts) (*forges.DeployKey, error) {
	return nil, m.createErr
}

func (m *mockDeployKeyService) Delete(context.Context, string, string, int64) error {
	return nil
}

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

func TestDeployKeyAddNotSupported(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"read-only", nil, "deploy keys is not supported by this forge"},
		{"write access", []string{"--read-only=false"}, "deploy keys with write access is not supported by this forge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := &mockDeployKeyService{createErr: fmt.Errorf("creating a deploy key: %w", forges.ErrNotSupported)}
			resolve.SetTestForge(&mockForge{deployKeyService: keys}, "owner", "repo", "example.com")
			t.Cleanup(resolve.ResetTestForge)

			cmd := deployKeyAddCmd()
			cmd.SetArgs(append([]string{"--title", "ci", "--key", "ssh-ed25519 AAAA"}, tt.args...))
			err := cmd.Execute()
			if err == nil || err.Error() != tt.want {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

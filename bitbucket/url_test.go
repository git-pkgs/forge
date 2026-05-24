package bitbucket

import "testing"

func TestParsePath(t *testing.T) {
	tests := []struct {
		name         string
		parts        []string
		wantOwner    string
		wantRepo     string
		wantResource string
		wantNumber   int
		wantErr      bool
	}{
		{
			name:      "repo only",
			parts:     []string{"owner", "repo"},
			wantOwner: "owner", wantRepo: "repo",
		},
		{
			name:      "pull request",
			parts:     []string{"owner", "repo", "pull-requests", "123"},
			wantOwner: "owner", wantRepo: "repo",
			wantResource: "pr", wantNumber: 123,
		},
		{
			name:      "issue",
			parts:     []string{"owner", "repo", "issues", "456"},
			wantOwner: "owner", wantRepo: "repo",
			wantResource: "issue", wantNumber: 456,
		},
		{
			name:    "missing repo",
			parts:   []string{"owner"},
			wantErr: true,
		},
		{
			name:    "invalid PR number",
			parts:   []string{"owner", "repo", "pull-requests", "abc"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, resource, number, err := parsePath(tt.parts)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tt.wantOwner {
				t.Errorf("owner: got %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo: got %q, want %q", repo, tt.wantRepo)
			}
			if resource != tt.wantResource {
				t.Errorf("resource: got %q, want %q", resource, tt.wantResource)
			}
			if number != tt.wantNumber {
				t.Errorf("number: got %d, want %d", number, tt.wantNumber)
			}
		})
	}
}

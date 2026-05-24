package gitlab

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
			parts:     []string{"group", "project"},
			wantOwner: "group", wantRepo: "project",
		},
		{
			name:      "nested group",
			parts:     []string{"group", "subgroup", "project"},
			wantOwner: "group/subgroup", wantRepo: "project",
		},
		{
			name:      "merge request",
			parts:     []string{"group", "project", "-", "merge_requests", "123"},
			wantOwner: "group", wantRepo: "project",
			wantResource: "pr", wantNumber: 123,
		},
		{
			name:      "nested group merge request",
			parts:     []string{"group", "subgroup", "project", "-", "merge_requests", "123"},
			wantOwner: "group/subgroup", wantRepo: "project",
			wantResource: "pr", wantNumber: 123,
		},
		{
			name:      "issue",
			parts:     []string{"group", "project", "-", "issues", "456"},
			wantOwner: "group", wantRepo: "project",
			wantResource: "issue", wantNumber: 456,
		},
		{
			name:    "missing repo",
			parts:   []string{"group"},
			wantErr: true,
		},
		{
			name:    "invalid MR number",
			parts:   []string{"group", "project", "-", "merge_requests", "abc"},
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

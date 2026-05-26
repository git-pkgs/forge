package gitlab

import (
	"testing"

	forges "github.com/git-pkgs/forge"
)

func TestParsePath(t *testing.T) {
	tests := []struct {
		name         string
		parts        []string
		wantOwner    string
		wantRepo     string
		wantResource forges.ResourceType
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
			wantResource: forges.ResourceTypePR, wantNumber: 123,
		},
		{
			name:      "nested group merge request",
			parts:     []string{"group", "subgroup", "project", "-", "merge_requests", "123"},
			wantOwner: "group/subgroup", wantRepo: "project",
			wantResource: forges.ResourceTypePR, wantNumber: 123,
		},
		{
			name:      "issue",
			parts:     []string{"group", "project", "-", "issues", "456"},
			wantOwner: "group", wantRepo: "project",
			wantResource: forges.ResourceTypeIssue, wantNumber: 456,
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

	f := &gitLabForge{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := f.ParsePath(tt.parts)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ref.Owner != tt.wantOwner {
				t.Errorf("owner: got %q, want %q", ref.Owner, tt.wantOwner)
			}
			if ref.Repo != tt.wantRepo {
				t.Errorf("repo: got %q, want %q", ref.Repo, tt.wantRepo)
			}
			if ref.Type != tt.wantResource {
				t.Errorf("resource: got %q, want %q", ref.Type, tt.wantResource)
			}
			if ref.Number != tt.wantNumber {
				t.Errorf("number: got %d, want %d", ref.Number, tt.wantNumber)
			}
		})
	}
}

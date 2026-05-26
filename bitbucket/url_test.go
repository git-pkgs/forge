package bitbucket

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
			parts:     []string{"owner", "repo"},
			wantOwner: "owner", wantRepo: "repo",
		},
		{
			name:      "pull request",
			parts:     []string{"owner", "repo", "pull-requests", "123"},
			wantOwner: "owner", wantRepo: "repo",
			wantResource: forges.ResourceTypePR, wantNumber: 123,
		},
		{
			name:      "issue",
			parts:     []string{"owner", "repo", "issues", "456"},
			wantOwner: "owner", wantRepo: "repo",
			wantResource: forges.ResourceTypeIssue, wantNumber: 456,
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

	f := &bitbucketForge{}
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

package forges

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const (
	testCommitSHA      = "8e8c483db84b4bee98b60c0593521ed34d9990e8"
	testCommitSHAUpper = "8E8C483DB84B4BEE98B60C0593521ED34D9990E8"
)

func TestIsFullCommitSHA(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want bool
	}{
		{name: "lowercase full SHA", ref: testCommitSHA, want: true},
		{name: "uppercase full SHA", ref: testCommitSHAUpper, want: true},
		{name: "mixed case full SHA", ref: "8e8C483db84B4bee98b60C0593521ed34d9990E8", want: true},
		{name: "abbreviated SHA", ref: "8e8c483", want: false},
		{name: "branch name", ref: "main", want: false},
		{name: "tag name", ref: "v4.2.1", want: false},
		{name: "empty", ref: "", want: false},
		{name: "too short by one", ref: testCommitSHA[1:], want: false},
		{name: "too long by one", ref: testCommitSHA + "0", want: false},
		{name: "right length with non-hex character", ref: strings.Repeat("g", FullCommitSHALength), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsFullCommitSHA(tt.ref); got != tt.want {
				t.Errorf("IsFullCommitSHA(%q) = %v, want %v", tt.ref, got, tt.want)
			}
		})
	}
}

func TestValidateCommitRef(t *testing.T) {
	tests := []struct {
		name             string
		owner, repo, ref string
		wantErr          bool
	}{
		{name: "all set", owner: "actions", repo: "checkout", ref: "v4", wantErr: false},
		{name: "empty owner", owner: "", repo: "checkout", ref: "v4", wantErr: true},
		{name: "empty repo", owner: "actions", repo: "", ref: "v4", wantErr: true},
		{name: "empty ref", owner: "actions", repo: "checkout", ref: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCommitRef(tt.owner, tt.repo, tt.ref)
			if tt.wantErr && !errors.Is(err, ErrCommitRefRequired) {
				t.Errorf("ValidateCommitRef() error = %v, want ErrCommitRefRequired", err)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateCommitRef() error = %v, want nil", err)
			}
		})
	}
}

func TestResolvedCommitSHA(t *testing.T) {
	t.Run("normalizes case and whitespace", func(t *testing.T) {
		got, err := ResolvedCommitSHA("actions", "checkout", "v4", " "+testCommitSHAUpper+"\n")
		if err != nil {
			t.Fatalf("ResolvedCommitSHA() error = %v", err)
		}
		if got != testCommitSHA {
			t.Errorf("ResolvedCommitSHA() = %q, want %q", got, testCommitSHA)
		}
	})

	t.Run("rejects empty SHA", func(t *testing.T) {
		if _, err := ResolvedCommitSHA("actions", "checkout", "v4", "  \n"); err == nil {
			t.Fatal("ResolvedCommitSHA() error = nil, want empty SHA error")
		}
	})

	t.Run("rejects abbreviated SHA", func(t *testing.T) {
		if _, err := ResolvedCommitSHA("actions", "checkout", "v4", "deadbeef"); err == nil {
			t.Fatal("ResolvedCommitSHA() error = nil, want invalid SHA error")
		}
	})
}

func TestCommitRefErrorUnwraps(t *testing.T) {
	err := CommitRefError("actions", "checkout", "missing", ErrNotFound)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("CommitRefError() = %v, want an error wrapping ErrNotFound", err)
	}
	for _, want := range []string{"actions/checkout", `"missing"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("CommitRefError() = %q, want it to mention %s", err.Error(), want)
		}
	}
}

func TestClientResolveCommitRoutes(t *testing.T) {
	mock := &mockForge{commitService: &mockCommitService{sha: testCommitSHA}}
	c := &Client{
		forges: map[string]Forge{"example.com": mock},
		tokens: make(map[string]string),
	}

	sha, err := c.ResolveCommit(context.Background(), "https://example.com/test/repo", "v1.0.0")
	if err != nil {
		t.Fatalf("ResolveCommit() error = %v", err)
	}
	if sha != testCommitSHA {
		t.Errorf("ResolveCommit() = %q, want %q", sha, testCommitSHA)
	}

	cs := mock.commitService
	if cs.lastOwner != "test" || cs.lastRepo != "repo" || cs.lastRef != "v1.0.0" {
		t.Errorf("backend received owner=%q repo=%q ref=%q, want test/repo at v1.0.0",
			cs.lastOwner, cs.lastRepo, cs.lastRef)
	}
}

func TestClientResolveCommitPropagatesBackendError(t *testing.T) {
	mock := &mockForge{commitService: &mockCommitService{err: ErrNotFound}}
	c := &Client{
		forges: map[string]Forge{"example.com": mock},
		tokens: make(map[string]string),
	}

	if _, err := c.ResolveCommit(context.Background(), "https://example.com/test/repo", "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("ResolveCommit() error = %v, want ErrNotFound", err)
	}
}

func TestClientResolveCommitUnregisteredDomain(t *testing.T) {
	c := NewClient()
	if _, err := c.ResolveCommit(context.Background(), "https://example.com/test/repo", "v1.0.0"); err == nil {
		t.Error("ResolveCommit() error = nil, want error for unregistered domain")
	}
}

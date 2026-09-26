//go:build tinygo

package gitlab_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	forge "github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/gitlab"
)

type unexpectedTransport struct{ t *testing.T }

func (u unexpectedTransport) RoundTrip(*http.Request) (*http.Response, error) {
	u.t.Error("unsupported GitLab operation made an HTTP request")
	return nil, errors.New("unexpected HTTP request")
}

func TestTinyGoClientRouting(t *testing.T) {
	transport := unexpectedTransport{t}
	original := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original })
	for _, hc := range []*http.Client{nil, {Transport: transport}} {
		backend := gitlab.New("https://gitlab.example.test/", "fixture-token", hc)
		client := forge.NewClient(forge.WithForge("gitlab.example.test", backend))
		repo, err := client.FetchRepository(context.Background(), "https://gitlab.example.test/group/subgroup/project")
		if repo != nil || !errors.Is(err, forge.ErrNotSupported) {
			t.Errorf("FetchRepository = %+v, %v; want ErrNotSupported", repo, err)
		}
		sha, err := client.ResolveCommit(context.Background(), "https://gitlab.example.test/group/subgroup/project", "main")
		if sha != "" || !errors.Is(err, forge.ErrNotSupported) {
			t.Errorf("ResolveCommit = %q, %v; want ErrNotSupported", sha, err)
		}
	}
}

func TestTinyGoServicesReturnUnsupported(t *testing.T) {
	f := gitlab.New("https://gitlab.example.test", "", &http.Client{Transport: unexpectedTransport{t}})
	ctx := context.Background()
	tests := []struct {
		name string
		run  func() error
	}{
		{"repository create", func() error { _, err := f.Repos().Create(ctx, forge.CreateRepoOpts{Name: "project"}); return err }},
		{"issues", func() error { _, err := f.Issues().Get(ctx, "group", "project", 12); return err }},
		{"pull requests", func() error { _, err := f.PullRequests().Get(ctx, "group", "project", 12); return err }},
		{"labels", func() error { _, err := f.Labels().List(ctx, "group", "project", forge.ListLabelOpts{}); return err }},
		{"milestones", func() error { _, err := f.Milestones().Get(ctx, "group", "project", 12); return err }},
		{"releases", func() error { _, err := f.Releases().Get(ctx, "group", "project", "v1.0.0"); return err }},
		{"CI", func() error { _, err := f.CI().GetRun(ctx, "group", "project", 12); return err }},
		{"branches", func() error { _, err := f.Branches().List(ctx, "group", "project", forge.ListBranchOpts{}); return err }},
		{"deploy keys", func() error { _, err := f.DeployKeys().Get(ctx, "group", "project", 12); return err }},
		{"secrets", func() error { _, err := f.Secrets().List(ctx, "group", "project", forge.ListSecretOpts{}); return err }},
		{"notifications", func() error { _, err := f.Notifications().Get(ctx, "12"); return err }},
		{"reviews", func() error {
			_, err := f.Reviews().List(ctx, "group", "project", 12, forge.ListReviewOpts{})
			return err
		}},
		{"files", func() error { _, err := f.Files().Get(ctx, "group", "project", "README.md", "main"); return err }},
		{"collaborators", func() error {
			_, err := f.Collaborators().List(ctx, "group", "project", forge.ListCollaboratorOpts{})
			return err
		}},
		{"commit statuses", func() error { _, err := f.CommitStatuses().List(ctx, "group", "project", "main"); return err }},
		{"commits", func() error { _, err := f.Commits().ResolveCommit(ctx, "group", "project", "main"); return err }},
		{"rate limit", func() error { _, err := f.GetRateLimit(ctx); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); !errors.Is(err, forge.ErrNotSupported) {
				t.Errorf("error = %v, want ErrNotSupported", err)
			}
		})
	}
}

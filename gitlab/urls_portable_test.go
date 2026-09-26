package gitlab_test

import (
	"testing"

	forge "github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/gitlab"
)

func TestGitLabPortableURLs(t *testing.T) {
	f := gitlab.New("https://gitlab.example.test/gitlab/", "", nil)
	api, ok := f.(forge.APIBaseURLProvider)
	if !ok || api.APIBaseURL() != "https://gitlab.example.test/gitlab/api/v4" {
		t.Fatalf("APIBaseURL provider = %#v", f)
	}
	ref, err := f.ParsePath([]string{"group", "subgroup", "project", "-", "merge_requests", "42"})
	if err != nil {
		t.Fatal(err)
	}
	if ref.Owner != "group/subgroup" || ref.Repo != "project" || ref.Type != forge.ResourceTypePR || ref.Number != 42 {
		t.Errorf("ParsePath = %+v", ref)
	}
	const repo = "https://gitlab.example.test/group/subgroup/project"
	urls := map[string]string{
		"/-/settings":          f.Repos().SettingsURL(repo),
		"/-/wikis":             f.Repos().WikiURL(repo),
		"/-/pipelines":         f.Repos().ActionsURL(repo),
		"/-/releases":          f.Repos().ReleasesURL(repo),
		"/-/blob/main/file.go": f.Repos().BlobURL(repo, "main", "file.go"),
		"/-/issues":            f.Issues().ListURL(repo),
		"/-/merge_requests":    f.PullRequests().ListURL(repo),
		"/-/labels":            f.Labels().ListURL(repo),
	}
	for suffix, got := range urls {
		if got != repo+suffix {
			t.Errorf("URL = %q, want %q", got, repo+suffix)
		}
	}
}

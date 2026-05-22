package github

import "testing"

func TestGitHubRepoBlobURL(t *testing.T) {
	s := &gitHubRepoService{}
	got := s.BlobURL("https://github.com/owner/repo", "main", "cmd/main.go")
	want := "https://github.com/owner/repo/blob/main/cmd/main.go"
	if got != want {
		t.Errorf("BlobURL = %q, want %q", got, want)
	}
}

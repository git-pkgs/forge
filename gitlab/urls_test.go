package gitlab

import "testing"

func TestGitLabRepoBlobURL(t *testing.T) {
	s := &gitLabRepoService{}
	got := s.BlobURL("https://gitlab.com/group/project", "main", "src/app.go")
	want := "https://gitlab.com/group/project/-/blob/main/src/app.go"
	if got != want {
		t.Errorf("BlobURL = %q, want %q", got, want)
	}
}

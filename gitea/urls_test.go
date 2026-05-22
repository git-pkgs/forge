package gitea

import "testing"

func TestGiteaRepoBlobURL(t *testing.T) {
	s := &giteaRepoService{}
	got := s.BlobURL("https://codeberg.org/owner/repo", "main", "README.md")
	want := "https://codeberg.org/owner/repo/src/branch/main/README.md"
	if got != want {
		t.Errorf("BlobURL = %q, want %q", got, want)
	}
}

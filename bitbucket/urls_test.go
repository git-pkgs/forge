package bitbucket

import "testing"

func TestBitbucketRepoBlobURL(t *testing.T) {
	s := &bitbucketRepoService{}
	got := s.BlobURL("https://bitbucket.org/owner/repo", "master", "README.md")
	want := "https://bitbucket.org/owner/repo/src/master/README.md"
	if got != want {
		t.Errorf("BlobURL = %q, want %q", got, want)
	}
}

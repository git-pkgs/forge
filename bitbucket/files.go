package bitbucket

import (
	"context"

	forge "github.com/git-pkgs/forge"
)

type bitbucketFileService struct{}

func (f *bitbucketForge) Files() forge.FileService {
	return &bitbucketFileService{}
}

func (s *bitbucketFileService) Get(_ context.Context, _, _, _, _ string) (*forge.FileContent, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketFileService) List(_ context.Context, _, _, _, _ string) ([]forge.FileEntry, error) {
	return nil, forge.ErrNotSupported
}

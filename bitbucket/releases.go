package bitbucket

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"io"
	"os"
)

type bitbucketReleaseService struct{}

func (f *bitbucketForge) Releases() forge.ReleaseService {
	return &bitbucketReleaseService{}
}

func (s *bitbucketReleaseService) List(_ context.Context, _, _ string, _ forge.ListReleaseOpts) ([]forge.Release, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketReleaseService) Get(_ context.Context, _, _, _ string) (*forge.Release, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketReleaseService) GetLatest(_ context.Context, _, _ string) (*forge.Release, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketReleaseService) Create(_ context.Context, _, _ string, _ forge.CreateReleaseOpts) (*forge.Release, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketReleaseService) Update(_ context.Context, _, _, _ string, _ forge.UpdateReleaseOpts) (*forge.Release, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketReleaseService) Delete(_ context.Context, _, _, _ string) error {
	return forge.ErrNotSupported
}

func (s *bitbucketReleaseService) UploadAsset(_ context.Context, _, _, _ string, _ *os.File) (*forge.ReleaseAsset, error) {
	return nil, forge.ErrNotSupported
}

func (s *bitbucketReleaseService) DownloadAsset(_ context.Context, _, _ string, _ int64) (io.ReadCloser, error) {
	return nil, forge.ErrNotSupported
}

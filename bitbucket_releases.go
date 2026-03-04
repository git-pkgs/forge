package forges

import (
	"context"
	"io"
	"os"
)

type bitbucketReleaseService struct{}

func (f *bitbucketForge) Releases() ReleaseService {
	return &bitbucketReleaseService{}
}

func (s *bitbucketReleaseService) List(_ context.Context, _, _ string, _ ListReleaseOpts) ([]Release, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketReleaseService) Get(_ context.Context, _, _, _ string) (*Release, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketReleaseService) GetLatest(_ context.Context, _, _ string) (*Release, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketReleaseService) Create(_ context.Context, _, _ string, _ CreateReleaseOpts) (*Release, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketReleaseService) Update(_ context.Context, _, _, _ string, _ UpdateReleaseOpts) (*Release, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketReleaseService) Delete(_ context.Context, _, _, _ string) error {
	return ErrNotSupported
}

func (s *bitbucketReleaseService) UploadAsset(_ context.Context, _, _, _ string, _ *os.File) (*ReleaseAsset, error) {
	return nil, ErrNotSupported
}

func (s *bitbucketReleaseService) DownloadAsset(_ context.Context, _, _ string, _ int64) (io.ReadCloser, error) {
	return nil, ErrNotSupported
}

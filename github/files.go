package github

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"github.com/google/go-github/v82/github"
)

type gitHubFileService struct {
	client *github.Client
}

func (f *gitHubForge) Files() forge.FileService {
	return &gitHubFileService{client: f.client}
}

func (s *gitHubFileService) Get(ctx context.Context, owner, repo, path, ref string) (*forge.FileContent, error) {
	opts := &github.RepositoryContentGetOptions{}
	if ref != "" {
		opts.Ref = ref
	}

	file, _, resp, err := s.client.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	if file == nil {
		return nil, forge.ErrNotFound
	}

	content, err := file.GetContent()
	if err != nil {
		return nil, err
	}

	return &forge.FileContent{
		Name:    file.GetName(),
		Path:    file.GetPath(),
		Content: []byte(content),
		SHA:     file.GetSHA(),
	}, nil
}

func (s *gitHubFileService) List(ctx context.Context, owner, repo, path, ref string) ([]forge.FileEntry, error) {
	opts := &github.RepositoryContentGetOptions{}
	if ref != "" {
		opts.Ref = ref
	}

	_, dir, resp, err := s.client.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	entries := make([]forge.FileEntry, len(dir))
	for i, d := range dir {
		entries[i] = forge.FileEntry{
			Name: d.GetName(),
			Path: d.GetPath(),
			Type: d.GetType(),
			Size: int64(d.GetSize()),
		}
	}
	return entries, nil
}

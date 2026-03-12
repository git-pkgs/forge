package gitea

import (
	"context"
	"encoding/base64"
	forge "github.com/git-pkgs/forge"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

type giteaFileService struct {
	client *gitea.Client
}

func (f *giteaForge) Files() forge.FileService {
	return &giteaFileService{client: f.client}
}

func (s *giteaFileService) Get(ctx context.Context, owner, repo, path, ref string) (*forge.FileContent, error) {
	cr, resp, err := s.client.GetContents(owner, repo, ref, path)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	var content []byte
	if cr.Content != nil {
		if cr.Encoding != nil && *cr.Encoding == "base64" {
			content, err = base64.StdEncoding.DecodeString(*cr.Content)
			if err != nil {
				return nil, err
			}
		} else {
			content = []byte(*cr.Content)
		}
	}

	return &forge.FileContent{
		Name:    cr.Name,
		Path:    cr.Path,
		Content: content,
		SHA:     cr.SHA,
	}, nil
}

func (s *giteaFileService) List(ctx context.Context, owner, repo, path, ref string) ([]forge.FileEntry, error) {
	items, resp, err := s.client.ListContents(owner, repo, ref, path)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	entries := make([]forge.FileEntry, len(items))
	for i, item := range items {
		entries[i] = forge.FileEntry{
			Name: item.Name,
			Path: item.Path,
			Type: item.Type,
			Size: item.Size,
		}
	}
	return entries, nil
}

package gitlab

import (
	"context"
	"encoding/base64"
	forge "github.com/git-pkgs/forge"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabFileService struct {
	client *gitlab.Client
}

func (f *gitLabForge) Files() forge.FileService {
	return &gitLabFileService{client: f.client}
}

func (s *gitLabFileService) Get(ctx context.Context, owner, repo, path, ref string) (*forge.FileContent, error) {
	pid := owner + "/" + repo
	opts := &gitlab.GetFileOptions{}
	if ref != "" {
		opts.Ref = gitlab.Ptr(ref)
	}

	file, resp, err := s.client.RepositoryFiles.GetFile(pid, path, opts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	var content []byte
	if file.Encoding == "base64" {
		content, err = base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			return nil, err
		}
	} else {
		content = []byte(file.Content)
	}

	return &forge.FileContent{
		Name:    file.FileName,
		Path:    file.FilePath,
		Content: content,
		SHA:     file.BlobID,
	}, nil
}

func (s *gitLabFileService) List(ctx context.Context, owner, repo, path, ref string) ([]forge.FileEntry, error) {
	pid := owner + "/" + repo
	opts := &gitlab.ListTreeOptions{
		Path: gitlab.Ptr(path),
	}
	if ref != "" {
		opts.Ref = gitlab.Ptr(ref)
	}

	nodes, resp, err := s.client.Repositories.ListTree(pid, opts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	entries := make([]forge.FileEntry, len(nodes))
	for i, n := range nodes {
		typ := n.Type
		switch typ {
		case "blob":
			typ = "file"
		case "tree":
			typ = "dir"
		}
		entries[i] = forge.FileEntry{
			Name: n.Name,
			Path: n.Path,
			Type: typ,
		}
	}
	return entries, nil
}

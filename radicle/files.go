package radicle

import (
	"context"
	"errors"
	"strings"

	forges "github.com/git-pkgs/forge"
)

type fileService struct {
	f *radicleForge
}

type blobResponse struct {
	Binary  bool   `json:"binary"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

type treeResponse struct {
	Entries []struct {
		Path string `json:"path"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	} `json:"entries"`
}

// pathSegments splits a repository file path into its non-empty segments.
func pathSegments(filePath string) []string {
	var out []string
	for _, s := range strings.Split(filePath, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (s *fileService) Get(ctx context.Context, owner, repo, filePath, ref string) (*forges.FileContent, error) {
	segments := pathSegments(filePath)
	if len(segments) == 0 {
		return nil, errors.New("file path is required")
	}
	info, err := s.f.repoInfo(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	sha, err := info.resolveRef(ref)
	if err != nil {
		return nil, err
	}

	var blob blobResponse
	u := s.f.repoPath(info.RID, append([]string{"blob", sha}, segments...)...)
	if err := s.f.getJSON(ctx, u, &blob); err != nil {
		return nil, err
	}
	content := []byte(blob.Content)
	if blob.Binary {
		// The JSON API leaves out the content of binary files.
		content, err = s.f.getRaw(ctx, s.f.rawPath(info.RID, sha, segments))
		if err != nil {
			return nil, err
		}
	}

	p := blob.Path
	if p == "" {
		p = strings.Join(segments, "/")
	}
	name := blob.Name
	if name == "" {
		name = segments[len(segments)-1]
	}
	return &forges.FileContent{Name: name, Path: p, Content: content}, nil
}

func (s *fileService) List(ctx context.Context, owner, repo, filePath, ref string) ([]forges.FileEntry, error) {
	info, err := s.f.repoInfo(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	sha, err := info.resolveRef(ref)
	if err != nil {
		return nil, err
	}

	segments := pathSegments(filePath)
	u := s.f.repoPath(info.RID, append([]string{"tree", sha}, segments...)...)
	if len(segments) == 0 {
		// The root tree is only served with a trailing slash.
		u += "/"
	}
	var tree treeResponse
	if err := s.f.getJSON(ctx, u, &tree); err != nil {
		return nil, err
	}

	entries := make([]forges.FileEntry, 0, len(tree.Entries))
	for _, e := range tree.Entries {
		p := e.Path
		if p == "" {
			p = strings.Join(append(segments, e.Name), "/")
		}
		entries = append(entries, forges.FileEntry{Name: e.Name, Path: p, Type: entryType(e.Kind)})
	}
	return entries, nil
}

func entryType(kind string) string {
	switch kind {
	case "tree":
		return "dir"
	case "blob":
		return "file"
	default:
		return kind
	}
}

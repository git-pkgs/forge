package tangled

import (
	"context"
	"net/url"
	"path"
	"strings"

	forges "github.com/git-pkgs/forge"
)

type fileService struct {
	f *tangledForge
}

func (s *fileService) Get(context.Context, string, string, string, string) (*forges.FileContent, error) {
	return nil, forges.ErrNotSupported
}

func (s *fileService) List(ctx context.Context, owner, repo, filePath, ref string) ([]forges.FileEntry, error) {
	repoDID, err := s.f.repoDID(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if ref == "" {
		ref = "HEAD"
	}

	params := url.Values{}
	params.Set("repo", repoDID)
	params.Set("ref", ref)
	params.Set("path", strings.Trim(filePath, "/"))

	var raw any
	if err := s.f.xrpc(ctx, xrpcGetTree, params, &raw); err != nil {
		return nil, err
	}

	items, _ := collection(raw, "tree", "entries", "values")
	entries := make([]forges.FileEntry, 0, len(items))
	for _, item := range items {
		v, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := stringField(v, "name")
		entryPath := strings.TrimPrefix(stringField(v, "path"), "/")
		if entryPath == "" {
			entryPath = strings.TrimPrefix(path.Join(filePath, name), "/")
		}
		entryType := stringField(v, "type")
		if entryType == "" {
			entryType = typeFromMode(stringField(v, "mode"))
		}
		entries = append(entries, forges.FileEntry{
			Name: name,
			Path: entryPath,
			Type: entryType,
			Size: int64Field(v, "size"),
		})
	}
	return entries, nil
}

func typeFromMode(mode string) string {
	switch {
	case strings.HasPrefix(mode, "04") || mode == "tree" || mode == "dir":
		return "dir"
	case mode == "symlink" || mode == "120000":
		return "symlink"
	default:
		return "file"
	}
}

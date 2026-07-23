package gerrit

import (
	"context"
	"net/http"
	"path"

	forge "github.com/git-pkgs/forge"
)

type gerritFileService struct {
	forge *gerritForge
}

func (f *gerritForge) Files() forge.FileService {
	return &gerritFileService{forge: f}
}

func (s *gerritFileService) Get(ctx context.Context, owner, repo, filePath, ref string) (*forge.FileContent, error) {
	project := projectName(owner, repo)
	if ref == "" {
		ref = "HEAD"
	}
	value, err := s.forge.doText(ctx, http.MethodGet, "/projects/"+encodeID(project)+"/branches/"+encodeID(ref)+"/files/"+encodeID(filePath)+"/content", nil, nil)
	if err != nil {
		return nil, err
	}
	content, err := decodeBase64Text(value)
	if err != nil {
		return nil, err
	}
	return &forge.FileContent{
		Name:    path.Base(filePath),
		Path:    filePath,
		Content: []byte(content),
	}, nil
}

func (s *gerritFileService) List(_ context.Context, _, _, _, _ string) ([]forge.FileEntry, error) {
	return nil, forge.ErrNotSupported
}

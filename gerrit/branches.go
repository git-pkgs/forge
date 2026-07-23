package gerrit

import (
	"context"
	"net/http"

	forge "github.com/git-pkgs/forge"
)

type gerritBranchService struct {
	forge *gerritForge
}

func (f *gerritForge) Branches() forge.BranchService {
	return &gerritBranchService{forge: f}
}

func (s *gerritBranchService) List(ctx context.Context, owner, repo string, opts forge.ListBranchOpts) ([]forge.Branch, error) {
	project := projectName(owner, repo)
	var infos []struct {
		Ref       string `json:"ref"`
		Revision  string `json:"revision"`
		CanDelete bool   `json:"can_delete"`
	}
	if err := s.forge.doJSON(ctx, http.MethodGet, "/projects/"+encodeID(project)+"/branches/", nil, nil, &infos); err != nil {
		return nil, err
	}

	branches := make([]forge.Branch, 0, len(infos))
	for _, info := range infos {
		branches = append(branches, forge.Branch{
			Name: trimRefPrefix(info.Ref),
			SHA:  info.Revision,
		})
		if opts.Limit > 0 && len(branches) >= opts.Limit {
			break
		}
	}
	return branches, nil
}

func (s *gerritBranchService) Create(ctx context.Context, owner, repo, name, from string) (*forge.Branch, error) {
	project := projectName(owner, repo)
	body := map[string]string{}
	if from != "" {
		body["revision"] = from
	}

	var info struct {
		Ref      string `json:"ref"`
		Revision string `json:"revision"`
	}
	if err := s.forge.doJSON(ctx, http.MethodPut, "/projects/"+encodeID(project)+"/branches/"+encodeID(name), nil, body, &info); err != nil {
		return nil, err
	}

	return &forge.Branch{Name: trimRefPrefix(info.Ref), SHA: info.Revision}, nil
}

func (s *gerritBranchService) Delete(ctx context.Context, owner, repo, name string) error {
	project := projectName(owner, repo)
	return s.forge.doJSON(ctx, http.MethodDelete, "/projects/"+encodeID(project)+"/branches/"+encodeID(name), nil, nil, nil)
}

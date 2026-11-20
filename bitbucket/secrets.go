package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	forge "github.com/git-pkgs/forge"
)

// bitbucketSecretService manages repository variables for Bitbucket
// Pipelines. Like GitLab's project variables, every variable is listed,
// whether or not it is secured, and Set always stores a secured variable.
type bitbucketSecretService struct {
	token      string
	httpClient *http.Client
}

func (f *bitbucketForge) Secrets() forge.SecretService {
	return &bitbucketSecretService{token: f.token, httpClient: f.httpClient}
}

type bbPipelineVariable struct {
	UUID string `json:"uuid"`
	Key  string `json:"key"`
}

// Bitbucket variables carry no timestamps, so only the name is set.
func convertBitbucketVariable(bb bbPipelineVariable) forge.Secret {
	return forge.Secret{Name: bb.Key}
}

func (s *bitbucketSecretService) repoService() *bitbucketRepoService {
	return &bitbucketRepoService{token: s.token, httpClient: s.httpClient}
}

func variablesURL(owner, repo string) string {
	return fmt.Sprintf("%s/repositories/%s/%s/pipelines_config/variables", bitbucketAPI, owner, repo)
}

// variableURL returns the URL of a single variable. Variable UUIDs are
// wrapped in braces, which have to be escaped in the path.
func variableURL(owner, repo, uuid string) string {
	return variablesURL(owner, repo) + "/" + url.PathEscape(uuid)
}

func (s *bitbucketSecretService) List(ctx context.Context, owner, repo string, opts forge.ListSecretOpts) ([]forge.Secret, error) {
	u := variablesURL(owner, repo) + "?" + pageQuery(opts.Page, opts.PerPage, opts.Limit)
	return listPages(ctx, s.repoService(), u, opts.Limit, convertBitbucketVariable)
}

// findVariable returns the UUID of the variable called name, or "" when
// there is none. Variables can only be addressed by UUID, so this walks the
// full list.
func (s *bitbucketSecretService) findVariable(ctx context.Context, owner, repo, name string) (string, error) {
	u := variablesURL(owner, repo) + "?" + pageQuery(0, 0, 0)
	vars, err := listPages(ctx, s.repoService(), u, 0, func(bb bbPipelineVariable) bbPipelineVariable { return bb })
	if err != nil {
		return "", err
	}
	for _, v := range vars {
		if v.Key == name {
			return v.UUID, nil
		}
	}
	return "", nil
}

// Set creates a secured variable, or updates the variable with that name and
// marks it secured if it exists.
func (s *bitbucketSecretService) Set(ctx context.Context, owner, repo string, opts forge.SetSecretOpts) error {
	uuid, err := s.findVariable(ctx, owner, repo, opts.Name)
	if err != nil {
		return err
	}

	body := map[string]any{
		"key":     opts.Name,
		"value":   opts.Value,
		"secured": true,
	}
	if uuid == "" {
		return s.repoService().doJSON(ctx, http.MethodPost, variablesURL(owner, repo), body, nil)
	}
	return s.repoService().doJSON(ctx, http.MethodPut, variableURL(owner, repo, uuid), body, nil)
}

func (s *bitbucketSecretService) Delete(ctx context.Context, owner, repo, name string) error {
	uuid, err := s.findVariable(ctx, owner, repo, name)
	if err != nil {
		return err
	}
	if uuid == "" {
		return forge.ErrNotFound
	}
	return s.repoService().doJSON(ctx, http.MethodDelete, variableURL(owner, repo, uuid), nil, nil)
}

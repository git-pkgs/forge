package gitlab

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"io"
	"net/http"
	"os"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabReleaseService struct {
	client *gitlab.Client
}

func (f *gitLabForge) Releases() forge.ReleaseService {
	return &gitLabReleaseService{client: f.client}
}

func convertGitLabRelease(r *gitlab.Release) forge.Release {
	result := forge.Release{
		TagName: r.TagName,
		Title:   r.Name,
		Body:    r.Description,
		HTMLURL: r.Links.Self,
	}

	if r.Author.Username != "" {
		result.Author = forge.User{
			Login:     r.Author.Username,
			AvatarURL: r.Author.AvatarURL,
			HTMLURL:   r.Author.WebURL,
		}
	}

	for _, s := range r.Assets.Sources {
		switch s.Format {
		case "tar.gz":
			result.TarballURL = s.URL
		case "zip":
			result.ZipballURL = s.URL
		}
	}
	for _, l := range r.Assets.Links {
		result.Assets = append(result.Assets, forge.ReleaseAsset{
			ID:          int64(l.ID),
			Name:        l.Name,
			DownloadURL: l.DirectAssetURL,
		})
	}

	if r.CreatedAt != nil {
		result.CreatedAt = *r.CreatedAt
	}
	if r.ReleasedAt != nil {
		result.PublishedAt = *r.ReleasedAt
	}

	return result
}

func (s *gitLabReleaseService) List(ctx context.Context, owner, repo string, opts forge.ListReleaseOpts) ([]forge.Release, error) {
	pid := owner + "/" + repo
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	glOpts := &gitlab.ListReleasesOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage), Page: int64(page)},
	}

	var all []forge.Release
	for {
		releases, resp, err := s.client.Releases.ListReleases(pid, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, r := range releases {
			all = append(all, convertGitLabRelease(r))
		}
		if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		glOpts.Page = int64(resp.NextPage)
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *gitLabReleaseService) Get(ctx context.Context, owner, repo, tag string) (*forge.Release, error) {
	pid := owner + "/" + repo
	r, resp, err := s.client.Releases.GetRelease(pid, tag)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabRelease(r)
	return &result, nil
}

func (s *gitLabReleaseService) GetLatest(ctx context.Context, owner, repo string) (*forge.Release, error) {
	pid := owner + "/" + repo
	// GitLab has no GetLatestRelease endpoint; fetch the first page sorted by default (newest first).
	glOpts := &gitlab.ListReleasesOptions{
		ListOptions: gitlab.ListOptions{PerPage: 1, Page: 1},
	}
	releases, resp, err := s.client.Releases.ListReleases(pid, glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	if len(releases) == 0 {
		return nil, forge.ErrNotFound
	}
	result := convertGitLabRelease(releases[0])
	return &result, nil
}

func (s *gitLabReleaseService) Create(ctx context.Context, owner, repo string, opts forge.CreateReleaseOpts) (*forge.Release, error) {
	pid := owner + "/" + repo
	glOpts := &gitlab.CreateReleaseOptions{
		TagName:     gitlab.Ptr(opts.TagName),
		Name:        gitlab.Ptr(opts.Title),
		Description: gitlab.Ptr(opts.Body),
	}
	if opts.Target != "" {
		glOpts.Ref = gitlab.Ptr(opts.Target)
	}

	r, resp, err := s.client.Releases.CreateRelease(pid, glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabRelease(r)
	return &result, nil
}

func (s *gitLabReleaseService) Update(ctx context.Context, owner, repo, tag string, opts forge.UpdateReleaseOpts) (*forge.Release, error) {
	pid := owner + "/" + repo
	glOpts := &gitlab.UpdateReleaseOptions{}
	changed := false

	if opts.Title != nil {
		glOpts.Name = opts.Title
		changed = true
	}
	if opts.Body != nil {
		glOpts.Description = opts.Body
		changed = true
	}

	if !changed {
		return s.Get(ctx, owner, repo, tag)
	}

	r, resp, err := s.client.Releases.UpdateRelease(pid, tag, glOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitLabRelease(r)
	return &result, nil
}

func (s *gitLabReleaseService) Delete(ctx context.Context, owner, repo, tag string) error {
	pid := owner + "/" + repo
	_, resp, err := s.client.Releases.DeleteRelease(pid, tag)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitLabReleaseService) UploadAsset(_ context.Context, _, _, _ string, _ *os.File) (*forge.ReleaseAsset, error) {
	return nil, forge.ErrNotSupported
}

func (s *gitLabReleaseService) DownloadAsset(_ context.Context, _, _ string, _ int64) (io.ReadCloser, error) {
	return nil, forge.ErrNotSupported
}

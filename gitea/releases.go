package gitea

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"code.gitea.io/sdk/gitea"
)

const latestReleasesPageSize = 10

type giteaReleaseService struct {
	client *gitea.Client
}

func (f *giteaForge) Releases() forge.ReleaseService {
	return &giteaReleaseService{client: f.client}
}

func convertGiteaRelease(r *gitea.Release) forge.Release {
	result := forge.Release{
		TagName:     r.TagName,
		Title:       r.Title,
		Body:        r.Note,
		Draft:       r.IsDraft,
		Prerelease:  r.IsPrerelease,
		Target:      r.Target,
		HTMLURL:     r.HTMLURL,
		TarballURL:  r.TarURL,
		ZipballURL:  r.ZipURL,
		CreatedAt:   r.CreatedAt,
		PublishedAt: r.PublishedAt,
	}

	if r.Publisher != nil {
		result.Author = forge.User{
			Login:     r.Publisher.UserName,
			AvatarURL: r.Publisher.AvatarURL,
		}
	}

	for _, a := range r.Attachments {
		result.Assets = append(result.Assets, forge.ReleaseAsset{
			ID:          a.ID,
			Name:        a.Name,
			Size:        int(a.Size),
			DownloadURL: a.DownloadURL,
			CreatedAt:   a.Created,
		})
	}

	return result
}

func (s *giteaReleaseService) List(ctx context.Context, owner, repo string, opts forge.ListReleaseOpts) ([]forge.Release, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	var all []forge.Release
	for {
		releases, resp, err := s.client.ListReleases(owner, repo, gitea.ListReleasesOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, r := range releases {
			all = append(all, convertGiteaRelease(r))
		}
		if len(releases) < perPage || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		page++
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

func (s *giteaReleaseService) Get(ctx context.Context, owner, repo, tag string) (*forge.Release, error) {
	r, resp, err := s.client.GetReleaseByTag(owner, repo, tag)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaRelease(r)
	return &result, nil
}

func (s *giteaReleaseService) GetLatest(ctx context.Context, owner, repo string) (*forge.Release, error) {
	// List releases and find the first non-draft, non-prerelease
	releases, resp, err := s.client.ListReleases(owner, repo, gitea.ListReleasesOptions{
		ListOptions: gitea.ListOptions{Page: 1, PageSize: latestReleasesPageSize},
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	for _, r := range releases {
		if !r.IsDraft && !r.IsPrerelease {
			result := convertGiteaRelease(r)
			return &result, nil
		}
	}
	if len(releases) > 0 {
		result := convertGiteaRelease(releases[0])
		return &result, nil
	}
	return nil, forge.ErrNotFound
}

func (s *giteaReleaseService) Create(ctx context.Context, owner, repo string, opts forge.CreateReleaseOpts) (*forge.Release, error) {
	gOpts := gitea.CreateReleaseOption{
		TagName:      opts.TagName,
		Title:        opts.Title,
		Note:         opts.Body,
		IsDraft:      opts.Draft,
		IsPrerelease: opts.Prerelease,
	}
	if opts.Target != "" {
		gOpts.Target = opts.Target
	}

	r, resp, err := s.client.CreateRelease(owner, repo, gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaRelease(r)
	return &result, nil
}

func (s *giteaReleaseService) Update(ctx context.Context, owner, repo, tag string, opts forge.UpdateReleaseOpts) (*forge.Release, error) {
	existing, resp, err := s.client.GetReleaseByTag(owner, repo, tag)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	gOpts := gitea.EditReleaseOption{}
	changed := false

	if opts.Title != nil {
		gOpts.Title = *opts.Title
		changed = true
	}
	if opts.Body != nil {
		gOpts.Note = *opts.Body
		changed = true
	}
	if opts.Draft != nil {
		gOpts.IsDraft = opts.Draft
		changed = true
	}
	if opts.Prerelease != nil {
		gOpts.IsPrerelease = opts.Prerelease
		changed = true
	}
	if opts.Target != nil {
		gOpts.Target = *opts.Target
		changed = true
	}
	if opts.TagName != nil {
		gOpts.TagName = *opts.TagName
		changed = true
	}

	if !changed {
		result := convertGiteaRelease(existing)
		return &result, nil
	}

	r, resp, err := s.client.EditRelease(owner, repo, existing.ID, gOpts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGiteaRelease(r)
	return &result, nil
}

func (s *giteaReleaseService) Delete(ctx context.Context, owner, repo, tag string) error {
	existing, resp, err := s.client.GetReleaseByTag(owner, repo, tag)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}

	resp, err = s.client.DeleteRelease(owner, repo, existing.ID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *giteaReleaseService) UploadAsset(ctx context.Context, owner, repo, tag string, file *os.File) (*forge.ReleaseAsset, error) {
	existing, resp, err := s.client.GetReleaseByTag(owner, repo, tag)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	name := filepath.Base(file.Name())
	a, resp, err := s.client.CreateReleaseAttachment(owner, repo, existing.ID, file, name)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	result := forge.ReleaseAsset{
		ID:          a.ID,
		Name:        a.Name,
		Size:        int(a.Size),
		DownloadURL: a.DownloadURL,
		CreatedAt:   a.Created,
	}
	return &result, nil
}

func (s *giteaReleaseService) DownloadAsset(_ context.Context, _, _ string, _ int64) (io.ReadCloser, error) {
	return nil, forge.ErrNotSupported
}

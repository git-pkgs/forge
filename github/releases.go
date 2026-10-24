package github

import (
	"context"
	forge "github.com/git-pkgs/forge"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/go-github/v82/github"
)

type gitHubReleaseService struct {
	client *github.Client
}

func (f *gitHubForge) Releases() forge.ReleaseService {
	return &gitHubReleaseService{client: f.client}
}

func convertGitHubRelease(r *github.RepositoryRelease) forge.Release {
	result := forge.Release{
		ID:         r.GetID(),
		TagName:    r.GetTagName(),
		Title:      r.GetName(),
		Body:       r.GetBody(),
		Draft:      r.GetDraft(),
		Prerelease: r.GetPrerelease(),
		Target:     r.GetTargetCommitish(),
		HTMLURL:    r.GetHTMLURL(),
		TarballURL: r.GetTarballURL(),
		ZipballURL: r.GetZipballURL(),
	}

	if u := r.GetAuthor(); u != nil {
		result.Author = forge.User{
			Login:     u.GetLogin(),
			AvatarURL: u.GetAvatarURL(),
			HTMLURL:   u.GetHTMLURL(),
		}
	}

	for _, a := range r.Assets {
		result.Assets = append(result.Assets, forge.ReleaseAsset{
			ID:            a.GetID(),
			Name:          a.GetName(),
			Size:          a.GetSize(),
			DownloadCount: a.GetDownloadCount(),
			DownloadURL:   a.GetBrowserDownloadURL(),
		})
		if t := a.GetCreatedAt(); !t.IsZero() {
			result.Assets[len(result.Assets)-1].CreatedAt = t.Time
		}
	}

	if t := r.GetCreatedAt(); !t.IsZero() {
		result.CreatedAt = t.Time
	}
	if t := r.GetPublishedAt(); !t.IsZero() {
		result.PublishedAt = t.Time
	}

	return result
}

func (s *gitHubReleaseService) List(ctx context.Context, owner, repo string, opts forge.ListReleaseOpts) ([]forge.Release, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	ghOpts := &github.ListOptions{PerPage: perPage, Page: page}

	var all []forge.Release
	for {
		releases, resp, err := s.client.Repositories.ListReleases(ctx, owner, repo, ghOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, forge.ErrNotFound
			}
			return nil, err
		}
		for _, r := range releases {
			all = append(all, convertGitHubRelease(r))
		}
		if resp.NextPage == 0 || (opts.Limit > 0 && len(all) >= opts.Limit) {
			break
		}
		ghOpts.Page = resp.NextPage
	}

	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

// Draft releases 404 by tag on GitHub, so fall back to treating tag as a release ID.
func (s *gitHubReleaseService) getReleaseByTagOrID(ctx context.Context, owner, repo, tag string) (*github.RepositoryRelease, error) {
	r, resp, err := s.client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err == nil {
		return r, nil
	}
	if resp == nil || resp.StatusCode != http.StatusNotFound {
		return nil, err
	}

	id, parseErr := strconv.ParseInt(tag, 10, 64)
	if parseErr != nil {
		return nil, forge.ErrNotFound
	}

	r, resp, err = s.client.Repositories.GetRelease(ctx, owner, repo, id)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	return r, nil
}

func (s *gitHubReleaseService) Get(ctx context.Context, owner, repo, tag string) (*forge.Release, error) {
	r, err := s.getReleaseByTagOrID(ctx, owner, repo, tag)
	if err != nil {
		return nil, err
	}
	result := convertGitHubRelease(r)
	return &result, nil
}

func (s *gitHubReleaseService) GetLatest(ctx context.Context, owner, repo string) (*forge.Release, error) {
	r, resp, err := s.client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubRelease(r)
	return &result, nil
}

func (s *gitHubReleaseService) Create(ctx context.Context, owner, repo string, opts forge.CreateReleaseOpts) (*forge.Release, error) {
	ghRelease := &github.RepositoryRelease{
		TagName:    &opts.TagName,
		Name:       &opts.Title,
		Body:       &opts.Body,
		Draft:      &opts.Draft,
		Prerelease: &opts.Prerelease,
	}
	if opts.Target != "" {
		ghRelease.TargetCommitish = &opts.Target
	}
	if opts.GenerateNotes {
		ghRelease.GenerateReleaseNotes = &opts.GenerateNotes
	}

	r, resp, err := s.client.Repositories.CreateRelease(ctx, owner, repo, ghRelease)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubRelease(r)
	return &result, nil
}

func (s *gitHubReleaseService) Update(ctx context.Context, owner, repo, tag string, opts forge.UpdateReleaseOpts) (*forge.Release, error) {
	// Get existing release to find its ID
	existing, err := s.getReleaseByTagOrID(ctx, owner, repo, tag)
	if err != nil {
		return nil, err
	}

	ghRelease := &github.RepositoryRelease{}
	changed := false

	if opts.TagName != nil {
		ghRelease.TagName = opts.TagName
		changed = true
	}
	if opts.Target != nil {
		ghRelease.TargetCommitish = opts.Target
		changed = true
	}
	if opts.Title != nil {
		ghRelease.Name = opts.Title
		changed = true
	}
	if opts.Body != nil {
		ghRelease.Body = opts.Body
		changed = true
	}
	if opts.Draft != nil {
		ghRelease.Draft = opts.Draft
		changed = true
	}
	if opts.Prerelease != nil {
		ghRelease.Prerelease = opts.Prerelease
		changed = true
	}

	if !changed {
		result := convertGitHubRelease(existing)
		return &result, nil
	}

	r, resp, err := s.client.Repositories.EditRelease(ctx, owner, repo, existing.GetID(), ghRelease)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}
	result := convertGitHubRelease(r)
	return &result, nil
}

func (s *gitHubReleaseService) Delete(ctx context.Context, owner, repo, tag string) error {
	existing, err := s.getReleaseByTagOrID(ctx, owner, repo, tag)
	if err != nil {
		return err
	}

	resp, err := s.client.Repositories.DeleteRelease(ctx, owner, repo, existing.GetID())
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return forge.ErrNotFound
		}
		return err
	}
	return nil
}

func (s *gitHubReleaseService) UploadAsset(ctx context.Context, owner, repo, tag string, file *os.File) (*forge.ReleaseAsset, error) {
	existing, err := s.getReleaseByTagOrID(ctx, owner, repo, tag)
	if err != nil {
		return nil, err
	}

	// UploadReleaseAsset dereferences opts (opts.MediaType), so a nil options
	// value panics inside go-github. Name the asset after the file like the
	// gitea service does.
	opts := &github.UploadOptions{Name: filepath.Base(file.Name())}
	a, resp, err := s.client.Repositories.UploadReleaseAsset(ctx, owner, repo, existing.GetID(), opts, file)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, forge.ErrNotFound
		}
		return nil, err
	}

	result := forge.ReleaseAsset{
		ID:            a.GetID(),
		Name:          a.GetName(),
		Size:          a.GetSize(),
		DownloadCount: a.GetDownloadCount(),
		DownloadURL:   a.GetBrowserDownloadURL(),
	}
	return &result, nil
}

func (s *gitHubReleaseService) DownloadAsset(ctx context.Context, owner, repo string, assetID int64) (io.ReadCloser, error) {
	rc, _, err := s.client.Repositories.DownloadReleaseAsset(ctx, owner, repo, assetID, http.DefaultClient)
	if err != nil {
		return nil, err
	}
	return rc, nil
}

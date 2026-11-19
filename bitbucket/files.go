package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"

	forge "github.com/git-pkgs/forge"
)

const maxFileSize = 32 << 20 // 32 MB

type bitbucketFileService struct {
	token      string
	httpClient *http.Client
}

func (f *bitbucketForge) Files() forge.FileService {
	return &bitbucketFileService{token: f.token, httpClient: f.httpClient}
}

// bbSrcEntry is a file or directory from the /src endpoint, either on its own
// (with ?format=meta) or as an entry in a directory listing.
type bbSrcEntry struct {
	Path       string   `json:"path"`
	Type       string   `json:"type"` // commit_file or commit_directory
	Size       int64    `json:"size"`
	Attributes []string `json:"attributes"` // link, executable, subrepository, binary, lfs
	Commit     struct {
		Hash string `json:"hash"`
	} `json:"commit"`
}

type bbSrcResponse struct {
	Values []bbSrcEntry `json:"values"`
	Next   string       `json:"next"`
}

func (s *bitbucketFileService) repoService() *bitbucketRepoService {
	return &bitbucketRepoService{token: s.token, httpClient: s.httpClient}
}

// pathSegments splits a repository file path into its segments. The path is
// cleaned against the repository root first, so empty, "." and ".." segments
// are resolved and can never climb out of the /src endpoint.
func pathSegments(filePath string) []string {
	clean := path.Clean("/" + filePath)
	if clean == "/" {
		return nil
	}
	return strings.Split(clean[1:], "/")
}

// srcURL returns the /src URL for the path segments at revision rev.
func srcURL(owner, repo, rev string, segments []string) string {
	parts := make([]string, 0, len(segments)+1)
	parts = append(parts, url.PathEscape(rev))
	for _, seg := range segments {
		parts = append(parts, url.PathEscape(seg))
	}
	return fmt.Sprintf("%s/repositories/%s/%s/src/%s", bitbucketAPI, owner, repo, strings.Join(parts, "/"))
}

// srcRevision returns the revision to put in a /src URL for ref. An empty ref
// means the main branch, which Bitbucket serves as HEAD. The /src endpoint
// takes the revision as a single path segment, so a branch or tag containing
// a slash is resolved to its commit first.
func (s *bitbucketFileService) srcRevision(ctx context.Context, owner, repo, ref string) (string, error) {
	switch {
	case ref == "":
		return "HEAD", nil
	case strings.Contains(ref, "/"):
		return resolveSlashedRef(ctx, s.repoService(), owner, repo, ref)
	default:
		return ref, nil
	}
}

func (s *bitbucketFileService) Get(ctx context.Context, owner, repo, filePath, ref string) (*forge.FileContent, error) {
	segments := pathSegments(filePath)
	if len(segments) == 0 {
		return nil, errors.New("file path is required")
	}
	rev, err := s.srcRevision(ctx, owner, repo, ref)
	if err != nil {
		return nil, err
	}

	// The /src endpoint answers a directory with a JSON listing and a file
	// with its raw content, so ask for the metadata first to tell them apart.
	rs := s.repoService()
	var meta bbSrcEntry
	if err := rs.getJSON(ctx, srcURL(owner, repo, rev, segments)+"?format=meta", &meta); err != nil {
		return nil, err
	}
	p := strings.Join(segments, "/")
	if meta.Type == "commit_directory" {
		return nil, fmt.Errorf("%s is a directory", p)
	}
	if meta.Size > maxFileSize {
		return nil, fmt.Errorf("%s is %d bytes, larger than the %d byte limit", p, meta.Size, maxFileSize)
	}
	// Read the content from the commit the metadata came from, so a branch
	// that moves in between can't mix up two versions of the file.
	if meta.Commit.Hash != "" {
		rev = meta.Commit.Hash
	}

	u := srcURL(owner, repo, rev, segments)
	resp, err := rs.send(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxFileSize {
		return nil, fmt.Errorf("%s is larger than the %d byte limit", p, maxFileSize)
	}

	return &forge.FileContent{
		Name:    path.Base(p),
		Path:    p,
		Content: content,
	}, nil
}

func (s *bitbucketFileService) List(ctx context.Context, owner, repo, filePath, ref string) ([]forge.FileEntry, error) {
	rev, err := s.srcRevision(ctx, owner, repo, ref)
	if err != nil {
		return nil, err
	}

	rs := s.repoService()
	segments := pathSegments(filePath)
	dir := strings.Join(segments, "/")
	// Directories, the root included, are only listed with a trailing slash.
	u := srcURL(owner, repo, rev, segments) + "/?pagelen=100"
	var entries []forge.FileEntry
	for u != "" {
		page, err := listPage(ctx, rs, u, dir)
		if err != nil {
			return nil, err
		}
		for _, e := range page.Values {
			entries = append(entries, forge.FileEntry{
				Name: path.Base(e.Path),
				Path: e.Path,
				Type: srcEntryType(e),
				Size: e.Size,
			})
		}
		u = page.Next
	}
	return entries, nil
}

// listPage fetches one page of the listing of dir. The /src endpoint serves a
// file's raw content rather than an error when dir names a file, which shows
// up as a response that isn't JSON.
func listPage(ctx context.Context, rs *bitbucketRepoService, u, dir string) (*bbSrcResponse, error) {
	resp, err := rs.send(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mediaType != "application/json" {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}
	var page bbSrcResponse
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}
	return &page, nil
}

func srcEntryType(e bbSrcEntry) string {
	switch {
	case e.Type == "commit_directory":
		return "dir"
	case slices.Contains(e.Attributes, "link"):
		return "symlink"
	case slices.Contains(e.Attributes, "subrepository"):
		return "submodule"
	default:
		return "file"
	}
}

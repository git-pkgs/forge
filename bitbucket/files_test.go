package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	forge "github.com/git-pkgs/forge"
)

const fileTestCommit = "5b6c39df3196eeeb8a36684beb9a3df854a0b5f3"

// newFileTestForge points the Bitbucket API at a test server running h.
func newFileTestForge(t *testing.T, h http.Handler) forge.Forge {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	t.Cleanup(func() { setBitbucketAPI(origAPI) })
	return New("test-token", srv.Client())
}

func writeSrcMeta(w http.ResponseWriter, path, typ string, size int64) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"path":   path,
		"type":   typ,
		"size":   size,
		"commit": map[string]any{"hash": fileTestCommit},
	})
}

func TestBitbucketGetFile(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/src/main/docs/README.md", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "meta" {
			t.Errorf("content requested at the ref instead of the resolved commit")
		}
		writeSrcMeta(w, "docs/README.md", "commit_file", 5)
	})
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/src/"+fileTestCommit+"/docs/README.md", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want bearer token", got)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("hello"))
	})

	f := newFileTestForge(t, mux)
	file, err := f.Files().Get(context.Background(), "atlassian", "stash", "docs/README.md", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Name", "README.md", file.Name)
	assertEqual(t, "Path", "docs/README.md", file.Path)
	assertEqual(t, "Content", "hello", string(file.Content))
}

func TestBitbucketGetFileDefaultsToHEAD(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/src/HEAD/FUNDING.yml", func(w http.ResponseWriter, _ *http.Request) {
		writeSrcMeta(w, "FUNDING.yml", "commit_file", 4)
	})
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/src/"+fileTestCommit+"/FUNDING.yml", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("x: y"))
	})

	f := newFileTestForge(t, mux)
	file, err := f.Files().Get(context.Background(), "atlassian", "stash", "FUNDING.yml", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "Content", "x: y", string(file.Content))
}

func TestBitbucketGetFileSlashedRef(t *testing.T) {
	f := newFileTestForge(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/2.0/repositories/atlassian/stash/refs/branches/feature%2Flogin":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"target": map[string]any{"hash": fileTestCommit},
			})
		case "/2.0/repositories/atlassian/stash/src/" + fileTestCommit + "/a.txt":
			if r.URL.Query().Get("format") == "meta" {
				writeSrcMeta(w, "a.txt", "commit_file", 1)
				return
			}
			_, _ = w.Write([]byte("a"))
		default:
			t.Errorf("unexpected request %s", r.URL.EscapedPath())
			http.NotFound(w, r)
		}
	}))

	file, err := f.Files().Get(context.Background(), "atlassian", "stash", "a.txt", "feature/login")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "Content", "a", string(file.Content))
}

func TestBitbucketGetFileDirectory(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/src/main/docs", func(w http.ResponseWriter, _ *http.Request) {
		writeSrcMeta(w, "docs", "commit_directory", 0)
	})

	f := newFileTestForge(t, mux)
	_, err := f.Files().Get(context.Background(), "atlassian", "stash", "docs", "main")
	if err == nil || !strings.Contains(err.Error(), "docs is a directory") {
		t.Fatalf("error = %v, want directory error", err)
	}
}

func TestBitbucketGetFileTooLarge(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/src/main/big.bin", func(w http.ResponseWriter, _ *http.Request) {
		writeSrcMeta(w, "big.bin", "commit_file", maxFileSize+1)
	})

	f := newFileTestForge(t, mux)
	_, err := f.Files().Get(context.Background(), "atlassian", "stash", "big.bin", "main")
	if err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("error = %v, want size limit error", err)
	}
}

func TestBitbucketGetFileNotFound(t *testing.T) {
	f := newFileTestForge(t, http.NotFoundHandler())
	_, err := f.Files().Get(context.Background(), "atlassian", "stash", "missing.txt", "main")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestBitbucketGetFileRequiresPath(t *testing.T) {
	f := New("", nil)
	for _, p := range []string{"", "/", ".", "a/.."} {
		if _, err := f.Files().Get(context.Background(), "atlassian", "stash", p, "main"); err == nil {
			t.Errorf("Get(%q): expected an error", p)
		}
	}
}

func TestBitbucketFilePathsStayInsideSrc(t *testing.T) {
	var paths []string
	f := newFileTestForge(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.EscapedPath())
		http.NotFound(w, r)
	}))

	ctx := context.Background()
	_, _ = f.Files().Get(ctx, "atlassian", "stash", "../../../pullrequests/1", "main")
	_, _ = f.Files().List(ctx, "atlassian", "stash", "docs/../../..", "main")

	want := []string{
		"/2.0/repositories/atlassian/stash/src/main/pullrequests/1",
		"/2.0/repositories/atlassian/stash/src/main/",
	}
	if !slices.Equal(paths, want) {
		t.Errorf("requested %v, want %v", paths, want)
	}
}

func TestSrcURLEscapesSegments(t *testing.T) {
	got := srcURL("o", "r", "v1.0", []string{"a b", "c#d?.txt"})
	want := bitbucketAPI + "/repositories/o/r/src/v1.0/a%20b/c%23d%3F.txt"
	if got != want {
		t.Errorf("srcURL = %q, want %q", got, want)
	}
}

func TestBitbucketListFiles(t *testing.T) {
	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/src/main/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if r.URL.Query().Get("page") == "" {
			if got := r.URL.Query().Get("pagelen"); got != "100" {
				t.Errorf("pagelen = %q, want 100", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"path": "docs", "type": "commit_directory"},
					{"path": "README.md", "type": "commit_file", "size": 120, "attributes": []string{}},
				},
				"next": srvURL + "/2.0/repositories/atlassian/stash/src/main/?pagelen=100&page=2",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{"path": "current", "type": "commit_file", "size": 7, "attributes": []string{"link"}},
				{"path": "vendor/lib", "type": "commit_file", "attributes": []string{"subrepository"}},
				{"path": "run.sh", "type": "commit_file", "size": 30, "attributes": []string{"executable"}},
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	defer setBitbucketAPI(origAPI)

	entries, err := New("", srv.Client()).Files().List(context.Background(), "atlassian", "stash", "", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []forge.FileEntry{
		{Name: "docs", Path: "docs", Type: "dir"},
		{Name: "README.md", Path: "README.md", Type: "file", Size: 120},
		{Name: "current", Path: "current", Type: "symlink", Size: 7},
		{Name: "lib", Path: "vendor/lib", Type: "submodule"},
		{Name: "run.sh", Path: "run.sh", Type: "file", Size: 30},
	}
	if !slices.Equal(entries, want) {
		t.Errorf("entries = %+v, want %+v", entries, want)
	}
}

func TestBitbucketListFilesSubdirectory(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/src/HEAD/docs/guides/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{"path": "docs/guides/setup.md", "type": "commit_file", "size": 10},
			},
		})
	})

	f := newFileTestForge(t, mux)
	entries, err := f.Files().List(context.Background(), "atlassian", "stash", "/docs/guides/", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []forge.FileEntry{{Name: "setup.md", Path: "docs/guides/setup.md", Type: "file", Size: 10}}
	if !slices.Equal(entries, want) {
		t.Errorf("entries = %+v, want %+v", entries, want)
	}
}

func TestBitbucketListFilesOnFile(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/src/main/package.json/", func(w http.ResponseWriter, _ *http.Request) {
		// Bitbucket serves the file itself, even for JSON files.
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(`{"values": []}`))
	})

	f := newFileTestForge(t, mux)
	_, err := f.Files().List(context.Background(), "atlassian", "stash", "package.json", "main")
	if err == nil || !strings.Contains(err.Error(), "package.json is not a directory") {
		t.Fatalf("error = %v, want not a directory error", err)
	}
}

func TestBitbucketListFilesSlashedRefNotFound(t *testing.T) {
	var paths []string
	f := newFileTestForge(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.EscapedPath())
		http.NotFound(w, r)
	}))

	_, err := f.Files().List(context.Background(), "atlassian", "stash", "", "release/1.x")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
	want := []string{
		"/2.0/repositories/atlassian/stash/refs/branches/release%2F1.x",
		"/2.0/repositories/atlassian/stash/refs/tags/release%2F1.x",
	}
	if !slices.Equal(paths, want) {
		t.Errorf("requested %v, want %v", paths, want)
	}
}

func TestBitbucketListFilesRejectsForeignNextLink(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/src/main/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"values": [], "next": "https://collector.example/capture"}`))
	})

	f := newFileTestForge(t, mux)
	_, err := f.Files().List(context.Background(), "atlassian", "stash", "", "main")
	if err == nil || !strings.Contains(err.Error(), "pagination link") {
		t.Errorf("error = %v, want a refused pagination link", err)
	}
}

package radicle

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	forges "github.com/git-pkgs/forge"
)

const (
	testRID    = "rad:z3gqcJUoA1n9HaHKufZs5FCSGazv5"
	testHead   = "e6fe52075382479165467933baa23dcfd185006e"
	testBranch = "7f5de25223c43ee90365fbbca85fc1bdfb41797f"
	testTag    = "d39ba83cfc7c7685548d18652fef039e25b19276"
	testLight  = "7b1847d8ad165599c177aa33a68b1fd98164def9"
)

const testRepoJSON = `{
  "rid": "rad:z3gqcJUoA1n9HaHKufZs5FCSGazv5",
  "payloads": {
    "xyz.radicle.project": {
      "data": {"name": "heartwood", "description": "Radicle Heartwood Protocol & Stack", "defaultBranch": "main"},
      "meta": {"head": "e6fe52075382479165467933baa23dcfd185006e", "issues": {"open": 190, "closed": 63}, "patches": {"open": 49}}
    }
  },
  "delegates": [{"id": "did:key:z6MksFqXN3Yhqk8pTJdUGLwATkRfQvwZXPqR2qMEhbS9wzpT", "alias": "cloudhead"}],
  "visibility": {"type": "public"},
  "refs": {
    "refs": {
      "refs/heads/main": "e6fe52075382479165467933baa23dcfd185006e",
      "refs/heads/releases/1": "7f5de25223c43ee90365fbbca85fc1bdfb41797f"
    },
    "tags": {
      "refs/tags/releases/1.0.0": {"commit": "d39ba83cfc7c7685548d18652fef039e25b19276", "message": "Release 1.0.0"},
      "refs/tags/light": "7b1847d8ad165599c177aa33a68b1fd98164def9"
    }
  }
}`

// newTestServer serves a radicle-httpd API for testRID. Each handler records
// requests it did not expect through t.Errorf.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	repo := "/api/v1/repos/" + testRID
	mux.HandleFunc("GET "+repo, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, testRepoJSON)
	})
	mux.HandleFunc("GET "+repo+"/tree/{sha}/{path...}", func(w http.ResponseWriter, r *http.Request) {
		sha, p := r.PathValue("sha"), r.PathValue("path")
		switch {
		case sha == testHead && p == "":
			_, _ = fmt.Fprint(w, `{"entries":[{"path":"crates","oid":"4f3f","name":"crates","kind":"tree"},{"path":"README.md","oid":"81f4","name":"README.md","kind":"blob"}],"name":"","path":""}`)
		case sha == testBranch && p == "crates":
			_, _ = fmt.Fprint(w, `{"entries":[{"path":"crates/radicle","oid":"6d22","name":"radicle","kind":"tree"},{"path":"crates/Cargo.toml","oid":"1a2b","name":"Cargo.toml","kind":"blob"}],"name":"crates","path":"crates"}`)
		default:
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("GET "+repo+"/blob/{sha}/{path...}", func(w http.ResponseWriter, r *http.Request) {
		sha, p := r.PathValue("sha"), r.PathValue("path")
		switch {
		case sha == testTag && p == "README.md":
			_, _ = fmt.Fprint(w, `{"binary":false,"name":"README.md","path":"README.md","content":"# heartwood\n"}`)
		case sha == testHead && p == "docs/logo.png":
			_, _ = fmt.Fprint(w, `{"binary":true,"name":"logo.png","path":"docs/logo.png"}`)
		case sha == testHead && p == "docs/file name.txt":
			_, _ = fmt.Fprint(w, `{"binary":false,"name":"file name.txt","path":"docs/file name.txt","content":"spaced"}`)
		default:
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("GET /raw/"+testRID+"/"+testHead+"/docs/logo.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 'P', 'N', 'G'})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestRepoGet(t *testing.T) {
	srv := newTestServer(t)
	f := New(srv.URL, "", srv.Client())

	repo, err := f.Repos().Get(context.Background(), "", testRID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if repo.FullName != testRID || repo.Owner != "" || repo.Name != "heartwood" {
		t.Errorf("identity = %q/%q/%q, want %q with no owner and name heartwood", repo.FullName, repo.Owner, repo.Name, testRID)
	}
	if repo.Description != "Radicle Heartwood Protocol & Stack" {
		t.Errorf("Description = %q", repo.Description)
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("DefaultBranch = %q, want main", repo.DefaultBranch)
	}
	if want := srv.URL + "/z3gqcJUoA1n9HaHKufZs5FCSGazv5.git"; repo.CloneURL != want {
		t.Errorf("CloneURL = %q, want %q", repo.CloneURL, want)
	}
	if repo.HTMLURL != "" {
		t.Errorf("HTMLURL = %q, want empty since radicle-httpd has no web interface", repo.HTMLURL)
	}
	if repo.OpenIssuesCount != 190 || repo.Private {
		t.Errorf("OpenIssuesCount = %d, Private = %v", repo.OpenIssuesCount, repo.Private)
	}
}

func TestRepoGetAcceptsRIDWithoutPrefix(t *testing.T) {
	srv := newTestServer(t)
	f := New(srv.URL, "", srv.Client())

	repo, err := f.Repos().Get(context.Background(), "", strings.TrimPrefix(testRID, RIDPrefix))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if repo.FullName != testRID {
		t.Errorf("FullName = %q, want %q", repo.FullName, testRID)
	}
}

func TestRepoGetRejectsInvalidInput(t *testing.T) {
	f := New("http://radicle.invalid", "", nil)
	tests := []struct{ name, owner, repo string }{
		{"owner given", "someone", testRID},
		{"empty", "", ""},
		{"prefix only", "", "rad:"},
		{"path traversal", "", "rad:z3gq/../../node"},
		{"query", "", "rad:z3gq?x=1"},
		{"escaped", "", "rad:z3gq%2F.."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := f.Repos().Get(context.Background(), tt.owner, tt.repo); err == nil {
				t.Fatalf("Get(%q, %q) succeeded, want error", tt.owner, tt.repo)
			}
		})
	}
}

func TestRepoGetNotFound(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)

	_, err := New(srv.URL, "", srv.Client()).Repos().Get(context.Background(), "", testRID)
	if !errors.Is(err, forges.ErrNotFound) {
		t.Fatalf("Get missing repo error = %v, want ErrNotFound", err)
	}
}

func TestRepoGetHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"Internal Server Error","code":500}`, http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	_, err := New(srv.URL, "", srv.Client()).Repos().Get(context.Background(), "", testRID)
	var httpErr *forges.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("error = %v, want HTTPError 500", err)
	}
}

func TestUnfollowedRedirectIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/raw/") {
			// A 3xx without Location is returned by http.Client as is.
			w.WriteHeader(http.StatusFound)
			_, _ = fmt.Fprint(w, "<html>moved</html>")
			return
		}
		if strings.Contains(r.URL.Path, "/blob/") {
			_, _ = fmt.Fprint(w, `{"binary":true,"name":"logo.png","path":"logo.png"}`)
			return
		}
		_, _ = fmt.Fprint(w, testRepoJSON)
	}))
	t.Cleanup(srv.Close)

	file, err := New(srv.URL, "", srv.Client()).Files().Get(context.Background(), "", testRID, "logo.png", "")
	var httpErr *forges.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusFound {
		t.Fatalf("Get = %+v, %v, want HTTPError 302", file, err)
	}
}

func TestNeverSendsToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("%s sent Authorization = %q, want none", r.URL.Path, got)
		}
		switch {
		case strings.Contains(r.URL.Path, "/tree/"):
			_, _ = fmt.Fprint(w, `{"entries":[]}`)
		case strings.Contains(r.URL.Path, "/blob/"):
			_, _ = fmt.Fprint(w, `{"binary":true,"name":"logo.png","path":"logo.png"}`)
		case strings.HasPrefix(r.URL.Path, "/raw/"):
			_, _ = w.Write([]byte("png"))
		default:
			_, _ = fmt.Fprint(w, testRepoJSON)
		}
	}))
	t.Cleanup(srv.Close)

	ctx := context.Background()
	f := New(srv.URL, "ghp_unrelated", srv.Client())
	if _, err := f.Repos().Get(ctx, "", testRID); err != nil {
		t.Fatalf("Repos().Get: %v", err)
	}
	if _, err := f.Files().List(ctx, "", testRID, "", ""); err != nil {
		t.Fatalf("Files().List: %v", err)
	}
	// A binary blob also exercises the raw endpoint.
	if _, err := f.Files().Get(ctx, "", testRID, "logo.png", ""); err != nil {
		t.Fatalf("Files().Get: %v", err)
	}
}

func TestBranchList(t *testing.T) {
	srv := newTestServer(t)
	f := New(srv.URL, "", srv.Client())

	branches, err := f.Branches().List(context.Background(), "", testRID, forges.ListBranchOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := []forges.Branch{
		{Name: "main", SHA: testHead, Default: true},
		{Name: "releases/1", SHA: testBranch},
	}
	if len(branches) != len(want) {
		t.Fatalf("got %d branches, want %d: %+v", len(branches), len(want), branches)
	}
	for i := range want {
		if branches[i] != want[i] {
			t.Errorf("branch %d = %+v, want %+v", i, branches[i], want[i])
		}
	}

	limited, err := f.Branches().List(context.Background(), "", testRID, forges.ListBranchOpts{Limit: 1})
	if err != nil {
		t.Fatalf("List with limit: %v", err)
	}
	if len(limited) != 1 || limited[0].Name != "main" {
		t.Errorf("limited branches = %+v, want only main", limited)
	}
}

func TestBranchListFallsBackToDefaultBranch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Older nodes report no canonical refs.
		_, _ = fmt.Fprint(w, `{"rid":"rad:z3gqcJUoA1n9HaHKufZs5FCSGazv5","payloads":{"xyz.radicle.project":{"data":{"name":"heartwood","defaultBranch":"main"},"meta":{"head":"e6fe52075382479165467933baa23dcfd185006e"}}}}`)
	}))
	t.Cleanup(srv.Close)

	branches, err := New(srv.URL, "", srv.Client()).Branches().List(context.Background(), "", testRID, forges.ListBranchOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(branches) != 1 || branches[0] != (forges.Branch{Name: "main", SHA: testHead, Default: true}) {
		t.Errorf("branches = %+v, want only the default branch", branches)
	}
}

func TestRepoListTags(t *testing.T) {
	srv := newTestServer(t)
	f := New(srv.URL, "", srv.Client())

	tags, err := f.Repos().ListTags(context.Background(), "", testRID)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	want := []forges.Tag{
		{Name: "light", Commit: testLight},
		{Name: "releases/1.0.0", Commit: testTag},
	}
	if len(tags) != len(want) {
		t.Fatalf("got %d tags, want %d: %+v", len(tags), len(want), tags)
	}
	for i := range want {
		if tags[i] != want[i] {
			t.Errorf("tag %d = %+v, want %+v", i, tags[i], want[i])
		}
	}
}

func TestFileListRootUsesDefaultHead(t *testing.T) {
	srv := newTestServer(t)
	f := New(srv.URL, "", srv.Client())

	entries, err := f.Files().List(context.Background(), "", testRID, "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := []forges.FileEntry{
		{Name: "crates", Path: "crates", Type: "dir"},
		{Name: "README.md", Path: "README.md", Type: "file"},
	}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(entries), len(want), entries)
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, entries[i], want[i])
		}
	}
}

func TestFileListResolvesBranch(t *testing.T) {
	srv := newTestServer(t)
	f := New(srv.URL, "", srv.Client())

	entries, err := f.Files().List(context.Background(), "", testRID, "/crates/", "releases/1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 || entries[0].Path != "crates/radicle" || entries[1].Type != "file" {
		t.Errorf("entries = %+v", entries)
	}
}

func TestFileGetResolvesTag(t *testing.T) {
	srv := newTestServer(t)
	f := New(srv.URL, "", srv.Client())

	file, err := f.Files().Get(context.Background(), "", testRID, "README.md", "releases/1.0.0")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if file.Name != "README.md" || file.Path != "README.md" || string(file.Content) != "# heartwood\n" {
		t.Errorf("file = %+v (content %q)", file, file.Content)
	}
}

func TestFileGetEscapesPath(t *testing.T) {
	srv := newTestServer(t)
	f := New(srv.URL, "", srv.Client())

	file, err := f.Files().Get(context.Background(), "", testRID, "docs/file name.txt", testHead)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(file.Content) != "spaced" {
		t.Errorf("content = %q, want spaced", file.Content)
	}
}

func TestFileGetBinaryUsesRawEndpoint(t *testing.T) {
	srv := newTestServer(t)
	f := New(srv.URL, "", srv.Client())

	file, err := f.Files().Get(context.Background(), "", testRID, "docs/logo.png", "HEAD")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(file.Content) != "\x89PNG" {
		t.Errorf("content = %q, want raw PNG bytes", file.Content)
	}
}

func TestPathSegments(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"/", ""},
		{"README.md", "README.md"},
		{"/crates//radicle/", "crates/radicle"},
		{"./docs/./file", "docs/file"},
		{"docs/../README.md", "README.md"},
		{"../../../nodes", "nodes"},
		{"..", ""},
	}
	for _, tt := range tests {
		if got := strings.Join(pathSegments(tt.in), "/"); got != tt.want {
			t.Errorf("pathSegments(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFileGetDotSegmentsStayInBlobEndpoint(t *testing.T) {
	srv := newTestServer(t)
	f := New(srv.URL, "", srv.Client())

	// Without cleaning, this would request /api/v1/repos/<rid>/blob/<sha>/../../../nodes
	// and could reach another endpoint. The server fails the test on any
	// request outside the handlers it registers.
	file, err := f.Files().Get(context.Background(), "", testRID, "../../docs/../README.md", "releases/1.0.0")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(file.Content) != "# heartwood\n" {
		t.Errorf("content = %q", file.Content)
	}
}

func TestFileGetRequiresPath(t *testing.T) {
	f := New("http://radicle.invalid", "", nil)
	for _, p := range []string{"", "/", "..", "./"} {
		if _, err := f.Files().Get(context.Background(), "", testRID, p, ""); err == nil {
			t.Errorf("Get(%q) succeeded, want error for empty file path", p)
		}
	}
}

func TestFileUnknownRefIsNotFound(t *testing.T) {
	srv := newTestServer(t)
	f := New(srv.URL, "", srv.Client())

	_, err := f.Files().List(context.Background(), "", testRID, "", "no-such-branch")
	if !errors.Is(err, forges.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestResolveRef(t *testing.T) {
	srv := newTestServer(t)
	info, err := New(srv.URL, "", srv.Client()).(*radicleForge).repoInfo(context.Background(), "", testRID)
	if err != nil {
		t.Fatalf("repoInfo: %v", err)
	}
	tests := []struct{ ref, want string }{
		{"", testHead},
		{"HEAD", testHead},
		{"main", testHead},
		{"refs/heads/releases/1", testBranch},
		{"releases/1", testBranch},
		{"releases/1.0.0", testTag},
		{"refs/tags/light", testLight},
		{strings.ToUpper(testBranch), testBranch},
	}
	for _, tt := range tests {
		got, err := info.resolveRef(tt.ref)
		if err != nil {
			t.Errorf("resolveRef(%q): %v", tt.ref, err)
			continue
		}
		if got != tt.want {
			t.Errorf("resolveRef(%q) = %q, want %q", tt.ref, got, tt.want)
		}
	}
}

func TestParsePath(t *testing.T) {
	f := New("https://seed.radicle.dev", "", nil)
	for _, path := range []string{
		testRID,
		testRID + "/tree/main",
		"nodes/seed.radicle.dev/" + testRID + "/issues/1e52373d",
	} {
		ref, err := f.ParsePath(strings.Split(path, "/"))
		if err != nil {
			t.Errorf("ParsePath(%q): %v", path, err)
			continue
		}
		if ref.Owner != "" || ref.Repo != testRID || ref.Number != 0 {
			t.Errorf("ParsePath(%q) = %+v", path, ref)
		}
	}
	if _, err := f.ParsePath([]string{"owner", "repo"}); err == nil {
		t.Error("ParsePath without a RID should fail")
	}
}

func TestAPIBaseURL(t *testing.T) {
	f := New("https://seed.radicle.dev/", "", nil).(forges.APIBaseURLProvider)
	if got := f.APIBaseURL(); got != "https://seed.radicle.dev/api/v1" {
		t.Errorf("APIBaseURL = %q", got)
	}
}

func TestUnsupportedServices(t *testing.T) {
	ctx := context.Background()
	f := New("http://radicle.invalid", "", nil)

	if _, err := f.Issues().Get(ctx, "", testRID, 1); !errors.Is(err, forges.ErrNotSupported) {
		t.Errorf("Issues().Get error = %v, want ErrNotSupported", err)
	}
	if _, err := f.PullRequests().List(ctx, "", testRID, forges.ListPROpts{}); !errors.Is(err, forges.ErrNotSupported) {
		t.Errorf("PullRequests().List error = %v, want ErrNotSupported", err)
	}
	if _, err := f.Branches().Create(ctx, "", testRID, "x", "main"); !errors.Is(err, forges.ErrNotSupported) {
		t.Errorf("Branches().Create error = %v, want ErrNotSupported", err)
	}
	if _, err := f.Repos().Create(ctx, forges.CreateRepoOpts{}); !errors.Is(err, forges.ErrNotSupported) {
		t.Errorf("Repos().Create error = %v, want ErrNotSupported", err)
	}
	if _, err := f.Commits().ResolveCommit(ctx, "", testRID, "main"); !errors.Is(err, forges.ErrNotSupported) {
		t.Errorf("Commits().ResolveCommit error = %v, want ErrNotSupported", err)
	}
	if _, err := f.GetAuthenticatedUser(ctx); !errors.Is(err, forges.ErrNotSupported) {
		t.Errorf("GetAuthenticatedUser error = %v, want ErrNotSupported", err)
	}
}

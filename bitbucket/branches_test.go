package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	forge "github.com/git-pkgs/forge"
)

const (
	branchTestMainSHA    = "5b6c39df3196eeeb8a36684beb9a3df854a0b5f3"
	branchTestFeatureSHA = "59a1e452abc40869b12cf13f476531bf5774d0d0"
)

// newBranchTestForge points the Bitbucket API at a test server running h.
func newBranchTestForge(t *testing.T, h http.Handler) (forge.Forge, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	origAPI := bitbucketAPI
	setBitbucketAPI(srv.URL + "/2.0")
	t.Cleanup(func() { setBitbucketAPI(origAPI) })
	return New("test-token", srv.Client()), srv
}

func branchJSON(name, hash string) map[string]any {
	return map[string]any{
		"name":   name,
		"target": map[string]any{"hash": hash},
	}
}

func TestBitbucketListBranches(t *testing.T) {
	var srvURL string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/refs/branches", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want bearer token", got)
		}
		if r.URL.Query().Get("page") == "" {
			if got := r.URL.Query().Get("pagelen"); got != "100" {
				t.Errorf("pagelen = %q, want 100", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []any{branchJSON("main", branchTestMainSHA)},
				"next":   srvURL + "/2.0/repositories/atlassian/stash/refs/branches?pagelen=100&page=2",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []any{branchJSON("feature/login", branchTestFeatureSHA)},
		})
	})
	f, srv := newBranchTestForge(t, mux)
	srvURL = srv.URL

	branches, err := f.Branches().List(context.Background(), "atlassian", "stash", forge.ListBranchOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []forge.Branch{
		{Name: "main", SHA: branchTestMainSHA},
		{Name: "feature/login", SHA: branchTestFeatureSHA},
	}
	if !slices.Equal(branches, want) {
		t.Errorf("branches = %+v, want %+v", branches, want)
	}
}

func TestBitbucketListBranchesLimit(t *testing.T) {
	var requests int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/refs/branches", func(w http.ResponseWriter, r *http.Request) {
		requests++
		if got := r.URL.Query().Get("pagelen"); got != "2" {
			t.Errorf("pagelen = %q, want 2", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []any{branchJSON("a", branchTestMainSHA), branchJSON("b", branchTestMainSHA)},
			"next":   "http://unused.invalid/next",
		})
	})
	f, _ := newBranchTestForge(t, mux)

	branches, err := f.Branches().List(context.Background(), "atlassian", "stash", forge.ListBranchOpts{Limit: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqualInt(t, "len", 2, len(branches))
	assertEqualInt(t, "requests", 1, requests)
}

func TestBitbucketListBranchesPaging(t *testing.T) {
	tests := []struct {
		name        string
		opts        forge.ListBranchOpts
		wantPageLen string
		wantPage    string
	}{
		{"per page capped", forge.ListBranchOpts{PerPage: 500}, "100", ""},
		{"per page", forge.ListBranchOpts{PerPage: 25}, "25", ""},
		{"start page", forge.ListBranchOpts{PerPage: 10, Page: 3}, "10", "3"},
		{"first page", forge.ListBranchOpts{Page: 1}, "100", ""},
		{"limit", forge.ListBranchOpts{Limit: 5}, "5", ""},
		{"limit with start page", forge.ListBranchOpts{PerPage: 10, Page: 3, Limit: 5}, "10", "3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("GET /2.0/repositories/atlassian/stash/refs/branches", func(w http.ResponseWriter, r *http.Request) {
				assertEqual(t, "pagelen", tt.wantPageLen, r.URL.Query().Get("pagelen"))
				assertEqual(t, "page", tt.wantPage, r.URL.Query().Get("page"))
				_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
			})
			f, _ := newBranchTestForge(t, mux)

			if _, err := f.Branches().List(context.Background(), "atlassian", "stash", tt.opts); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestBitbucketListBranchesNotFound(t *testing.T) {
	f, _ := newBranchTestForge(t, http.NotFoundHandler())
	_, err := f.Branches().List(context.Background(), "atlassian", "missing", forge.ListBranchOpts{})
	if !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

// bbCreateBranchBody is the request body of a branch creation.
type bbCreateBranchBody struct {
	Name   string `json:"name"`
	Target struct {
		Hash string `json:"hash"`
	} `json:"target"`
}

func TestBitbucketCreateBranch(t *testing.T) {
	var created bbCreateBranchBody
	f, _ := newBranchTestForge(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.EscapedPath() {
		case "GET /2.0/repositories/atlassian/stash/refs/branches/release%2F1.x":
			_ = json.NewEncoder(w).Encode(branchJSON("release/1.x", branchTestMainSHA))
		case "POST /2.0/repositories/atlassian/stash/refs/branches":
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
				t.Errorf("decoding body: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(branchJSON("feature/login", branchTestMainSHA))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.EscapedPath())
			http.NotFound(w, r)
		}
	}))

	branch, err := f.Branches().Create(context.Background(), "atlassian", "stash", "feature/login", "release/1.x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "Name", "feature/login", branch.Name)
	assertEqual(t, "SHA", branchTestMainSHA, branch.SHA)
	assertEqual(t, "body name", "feature/login", created.Name)
	assertEqual(t, "body target hash", branchTestMainSHA, created.Target.Hash)
}

func TestBitbucketCreateBranchFromCommit(t *testing.T) {
	var target string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /2.0/repositories/atlassian/stash/refs/branches", func(w http.ResponseWriter, r *http.Request) {
		var body bbCreateBranchBody
		_ = json.NewDecoder(r.Body).Decode(&body)
		target = body.Target.Hash
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(branchJSON("hotfix", body.Target.Hash))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})
	f, _ := newBranchTestForge(t, mux)

	branch, err := f.Branches().Create(context.Background(), "atlassian", "stash", "hotfix", branchTestFeatureSHA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "target", branchTestFeatureSHA, target)
	assertEqual(t, "SHA", branchTestFeatureSHA, branch.SHA)
}

func TestBitbucketCreateBranchSourceNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash/refs/branches/", http.NotFound)
	mux.HandleFunc("POST /2.0/repositories/atlassian/stash/refs/branches", func(w http.ResponseWriter, _ *http.Request) {
		t.Error("branch created from a source that doesn't exist")
	})
	f, _ := newBranchTestForge(t, mux)

	_, err := f.Branches().Create(context.Background(), "atlassian", "stash", "feature", "missing")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestBitbucketCreateBranchRejected(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /2.0/repositories/atlassian/stash/refs/branches", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type": "error", "error": {"message": "BRANCH_ALREADY_EXISTS"}}`))
	})
	f, _ := newBranchTestForge(t, mux)

	_, err := f.Branches().Create(context.Background(), "atlassian", "stash", "main", branchTestMainSHA)
	var httpErr *forge.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusBadRequest {
		t.Errorf("error = %v, want HTTP 400", err)
	}
}

func TestBitbucketDeleteBranch(t *testing.T) {
	var deleted string
	f, _ := newBranchTestForge(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		deleted = r.URL.EscapedPath()
		w.WriteHeader(http.StatusNoContent)
	}))

	if err := f.Branches().Delete(context.Background(), "atlassian", "stash", "feature/login"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEqual(t, "path", "/2.0/repositories/atlassian/stash/refs/branches/feature%2Flogin", deleted)
}

func TestBitbucketDeleteBranchNotFound(t *testing.T) {
	f, _ := newBranchTestForge(t, http.NotFoundHandler())
	err := f.Branches().Delete(context.Background(), "atlassian", "stash", "missing")
	if !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestBitbucketBranchWritesBlockedReadOnly(t *testing.T) {
	f := New("test-token", forge.ReadOnlyHTTPClient(nil))
	ctx := context.Background()

	if _, err := f.Branches().Create(ctx, "atlassian", "stash", "feature", branchTestMainSHA); !errors.Is(err, forge.ErrReadOnly) {
		t.Errorf("Create: err = %v, want ErrReadOnly", err)
	}
	if err := f.Branches().Delete(ctx, "atlassian", "stash", "feature"); !errors.Is(err, forge.ErrReadOnly) {
		t.Errorf("Delete: err = %v, want ErrReadOnly", err)
	}
}

package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/git-pkgs/forge/internal/config"
)

const (
	cliTestRID  = "rad:z3gqcJUoA1n9HaHKufZs5FCSGazv5"
	cliTestHead = "e6fe52075382479165467933baa23dcfd185006e"
)

// radicleCLIServer serves just enough of radicle-httpd for the CLI tests.
func radicleCLIServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := "/api/v1/repos/" + cliTestRID
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+repo, func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"rid":%q,"payloads":{"xyz.radicle.project":{"data":{"name":"heartwood","description":"Radicle Heartwood","defaultBranch":"main"},"meta":{"head":%q}}},"refs":{"refs":{"refs/heads/main":%q},"tags":{}}}`,
			cliTestRID, cliTestHead, cliTestHead)
	})
	mux.HandleFunc("GET "+repo+"/tree/"+cliTestHead+"/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"entries":[{"path":"crates","name":"crates","kind":"tree"},{"path":"README.md","name":"README.md","kind":"blob"}]}`)
	})
	mux.HandleFunc("GET "+repo+"/blob/"+cliTestHead+"/README.md", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"binary":false,"name":"README.md","path":"README.md","content":"# heartwood\n"}`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func runRadicleCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetCmd(rootCmd)
	t.Cleanup(func() {
		resetCmd(rootCmd)
		rootCmd.SetArgs(nil)
	})
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("FORGE_HOST", "")
	config.ResetCache()
	t.Cleanup(config.ResetCache)

	rootCmd.SetArgs(args)
	return captureStdout(t, rootCmd.Execute)
}

func TestRadicleRepoFlagFileCommands(t *testing.T) {
	srv := radicleCLIServer(t)
	repoFlag := srv.URL + "/" + cliTestRID

	stdout, err := runRadicleCLI(t, "-R", repoFlag, "--output", "plain", "file", "ls")
	if err != nil {
		t.Fatalf("file ls: %v", err)
	}
	if stdout != "crates\nREADME.md\n" {
		t.Errorf("file ls output = %q", stdout)
	}

	stdout, err = runRadicleCLI(t, "-R", repoFlag, "file", "cat", "README.md")
	if err != nil {
		t.Fatalf("file cat: %v", err)
	}
	if stdout != "# heartwood\n" {
		t.Errorf("file cat output = %q", stdout)
	}
}

func TestRadicleRepoView(t *testing.T) {
	srv := radicleCLIServer(t)

	stdout, err := runRadicleCLI(t, "-R", srv.URL+"/"+cliTestRID, "--output", "json", "repo", "view")
	if err != nil {
		t.Fatalf("repo view: %v", err)
	}
	if !strings.Contains(stdout, `"full_name": "`+cliTestRID+`"`) || !strings.Contains(stdout, `"default_branch": "main"`) {
		t.Errorf("repo view output = %s", stdout)
	}
}

func TestRadicleBrowseHasNoWebInterface(t *testing.T) {
	srv := radicleCLIServer(t)

	_, err := runRadicleCLI(t, "-R", srv.URL+"/"+cliTestRID, "browse", "--no-browser")
	if err == nil || !strings.Contains(err.Error(), "browse is not supported for "+cliTestRID) {
		t.Fatalf("browse error = %v, want unsupported error", err)
	}
}

func TestRepoDisplayName(t *testing.T) {
	if got := repoDisplayName("owner", "repo"); got != "owner/repo" {
		t.Errorf("repoDisplayName(owner, repo) = %q", got)
	}
	if got := repoDisplayName("", cliTestRID); got != cliTestRID {
		t.Errorf("repoDisplayName with empty owner = %q, want %q", got, cliTestRID)
	}
}

package funding

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"

	forges "github.com/git-pkgs/forge"
)

// fakeFiles answers FileService.Get from a map keyed by "owner/repo/path@ref",
// so a test can say exactly which funding file exists where.
type fakeFiles struct {
	files map[string]string
	errs  map[string]error
	calls []string
}

func (f *fakeFiles) key(owner, repo, path, ref string) string {
	return owner + "/" + repo + "/" + path + "@" + ref
}

func (f *fakeFiles) Get(_ context.Context, owner, repo, path, ref string) (*forges.FileContent, error) {
	key := f.key(owner, repo, path, ref)
	f.calls = append(f.calls, key)
	if err, ok := f.errs[key]; ok {
		return nil, err
	}
	body, ok := f.files[key]
	if !ok {
		return nil, forges.ErrNotFound
	}
	return &forges.FileContent{Path: path, Content: []byte(body)}, nil
}

func (f *fakeFiles) List(_ context.Context, _, _, _, _ string) ([]forges.FileEntry, error) {
	return nil, forges.ErrNotSupported
}

func TestParseReadsEveryPlatform(t *testing.T) {
	content := `# Funding for this project
github: [alice, bob]
patreon: alice
open_collective: "my-collective"
ko_fi: 'alice'
tidelift: npm/left-pad
community_bridge: my-project
liberapay: alice
issuehunt: alice
lfx_crowdfunding: my-project
polar: alice
buy_me_a_coffee: alice
thanks_dev: gh/alice
custom: ["https://example.com/donate", "https://example.org/#giving"]
otechie: alice
unknown_platform: ignored
`
	links, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := Links{
		GitHub:          []string{"alice", "bob"},
		Patreon:         "alice",
		OpenCollective:  "my-collective",
		KoFi:            "alice",
		Tidelift:        "npm/left-pad",
		CommunityBridge: "my-project",
		Liberapay:       "alice",
		IssueHunt:       "alice",
		LFXCrowdfunding: "my-project",
		Polar:           "alice",
		BuyMeACoffee:    "alice",
		ThanksDev:       "gh/alice",
		Otechie:         "alice",
		Custom:          []string{"https://example.com/donate", "https://example.org/#giving"},
	}
	if !reflect.DeepEqual(links, want) {
		t.Errorf("Parse:\n got %+v\nwant %+v", links, want)
	}
	if links.Empty() {
		t.Error("Empty() = true for a file that declares funding")
	}
}

func TestParseReadsBlockListsAndScalars(t *testing.T) {
	content := `github:
  - alice
  - bob
custom:
  - https://example.com/donate
  - https://example.org/docs#donate
patreon: alice  # a trailing comment
`
	links, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !reflect.DeepEqual(links.GitHub, []string{"alice", "bob"}) {
		t.Errorf("GitHub = %v", links.GitHub)
	}
	// A "#" inside a URL is not a comment, so the fragment survives.
	if !reflect.DeepEqual(links.Custom, []string{"https://example.com/donate", "https://example.org/docs#donate"}) {
		t.Errorf("Custom = %v", links.Custom)
	}
	if links.Patreon != "alice" {
		t.Errorf("Patreon = %q", links.Patreon)
	}
}

func TestParseErrorsOnMalformedLines(t *testing.T) {
	for name, content := range map[string]string{
		"list item without a key": "- alice\n",
		"line without a colon":    "github\n",
		"unterminated list":       "github: [alice, bob\n",
		"empty list item":         "github: [alice, , bob]\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(content)); err == nil {
				t.Errorf("Parse(%q) returned no error", content)
			}
		})
	}
}

func TestParseSkipsABOMAndADocumentMarker(t *testing.T) {
	content := "\ufeff---\ngithub: alice\n...\n"
	links, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !reflect.DeepEqual(links.GitHub, []string{"alice"}) {
		t.Errorf("GitHub = %v, want [alice]", links.GitHub)
	}
}

func TestParseHandlesCRLF(t *testing.T) {
	content := "github: [alice, bob]\r\npatreon: alice\r\n"
	links, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !reflect.DeepEqual(links.GitHub, []string{"alice", "bob"}) || links.Patreon != "alice" {
		t.Errorf("Parse = %+v", links)
	}
}

func TestParseEmptyContentIsEmpty(t *testing.T) {
	links, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !links.Empty() {
		t.Errorf("Parse(nil) = %+v, want empty", links)
	}
}

func TestURL(t *testing.T) {
	links := Links{}
	for platform, want := range map[string]string{
		"github":           "https://github.com/sponsors/alice",
		"patreon":          "https://www.patreon.com/alice",
		"open_collective":  "https://opencollective.com/alice",
		"ko_fi":            "https://ko-fi.com/alice",
		"tidelift":         "https://tidelift.com/funding/github/alice",
		"community_bridge": "https://funding.communitybridge.org/projects/alice",
		"liberapay":        "https://liberapay.com/alice",
		"issuehunt":        "https://issuehunt.io/r/alice",
		"lfx_crowdfunding": "https://crowdfunding.lfx.linuxfoundation.org/projects/alice",
		"polar":            "https://polar.sh/alice",
		"buy_me_a_coffee":  "https://www.buymeacoffee.com/alice",
		"thanks_dev":       "https://thanks.dev/alice",
	} {
		if got := links.URL(platform, "alice"); got != want {
			t.Errorf("URL(%q) = %q, want %q", platform, got, want)
		}
	}
	// Custom holds URLs already, so there is nothing to build.
	if got := links.URL("custom", "https://example.com"); got != "" {
		t.Errorf("URL(custom) = %q, want empty", got)
	}
}

func TestDiscoverPrefersTheGitHubDirectory(t *testing.T) {
	files := &fakeFiles{files: map[string]string{
		"alice/project/.github/FUNDING.yml@main": "github: alice\n",
		"alice/project/FUNDING.yml@main":         "patreon: alice\n",
	}}
	links, err := Discover(context.Background(), files, "alice", "project", "main")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if !reflect.DeepEqual(links.GitHub, []string{"alice"}) || links.Patreon != "" {
		t.Errorf("Discover = %+v, want the .github directory file", links)
	}
	if len(files.calls) != 1 {
		t.Errorf("Discover made %d calls, want 1: %v", len(files.calls), files.calls)
	}
}

func TestDiscoverFallsBackInGitHubsOrder(t *testing.T) {
	cases := map[string]struct {
		exists string
		want   string
		calls  int
	}{
		"github dir":    {".github/FUNDING.yml", "github: a", 1},
		"root":          {"FUNDING.yml", "github: a", 2},
		"docs":          {"docs/FUNDING.yml", "github: a", 3},
		"owner .github": {"OWNER:.github/FUNDING.yml", "github: a", 4},
		"owner root":    {"OWNER:FUNDING.yml", "github: a", 5},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			key := "alice/project/" + tc.exists + "@main"
			if handle, ok := strings.CutPrefix(tc.exists, "OWNER:"); ok {
				// The owner's .github repository is read from its own default
				// branch: a ref from owner/repo does not exist there.
				key = "alice/.github/" + handle + "@"
			}
			files := &fakeFiles{files: map[string]string{key: tc.want}}
			links, err := Discover(context.Background(), files, "alice", "project", "main")
			if err != nil {
				t.Fatalf("Discover: %v", err)
			}
			if !reflect.DeepEqual(links.GitHub, []string{"a"}) {
				t.Errorf("Discover = %+v, want the %s file", links, name)
			}
			if len(files.calls) != tc.calls {
				t.Errorf("Discover made %d calls, want %d: %v", len(files.calls), tc.calls, files.calls)
			}
		})
	}
}

func TestDiscoverReturnsEmptyWhenNothingIsFunded(t *testing.T) {
	files := &fakeFiles{}
	links, err := Discover(context.Background(), files, "alice", "project", "")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if !links.Empty() {
		t.Errorf("Discover = %+v, want empty", links)
	}
	// With an empty ref the default branch is read, which is how the call is
	// forwarded to the forge.
	for _, call := range files.calls {
		if !strings.HasSuffix(call, "@") {
			t.Errorf("call %q does not pass the empty ref through", call)
		}
	}
}

func TestDiscoverStopsAtAFileThatDeclaresNothing(t *testing.T) {
	// Existence decides which file GitHub reads, so a file with a key this
	// package does not know stops the walk instead of falling through to docs/.
	files := &fakeFiles{files: map[string]string{
		"alice/project/FUNDING.yml@main":      "unknown_platform: alice\n",
		"alice/project/docs/FUNDING.yml@main": "github: alice\n",
	}}
	links, err := Discover(context.Background(), files, "alice", "project", "main")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if !links.Empty() {
		t.Errorf("Discover = %+v, want empty", links)
	}
	if len(files.calls) != 2 {
		t.Errorf("Discover made %d calls, want 2: %v", len(files.calls), files.calls)
	}
}

func TestDiscoverTreatsAnUnsupportedForgeAsMissing(t *testing.T) {
	files := &fakeFiles{errs: map[string]error{
		"alice/project/.github/FUNDING.yml@main": forges.ErrNotSupported,
		"alice/project/FUNDING.yml@main":         forges.ErrNotSupported,
		"alice/project/docs/FUNDING.yml@main":    forges.ErrNotSupported,
		"alice/.github/.github/FUNDING.yml@":     forges.ErrNotSupported,
		"alice/.github/FUNDING.yml@":             forges.ErrNotSupported,
	}}
	links, err := Discover(context.Background(), files, "alice", "project", "main")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if !links.Empty() {
		t.Errorf("Discover = %+v, want empty", links)
	}
}

func TestDiscoverReturnsOtherErrors(t *testing.T) {
	files := &fakeFiles{errs: map[string]error{
		"alice/project/.github/FUNDING.yml@main": &forges.HTTPError{StatusCode: http.StatusForbidden},
	}}
	if _, err := Discover(context.Background(), files, "alice", "project", "main"); err == nil {
		t.Fatal("Discover returned no error for a 403")
	}

	// A 404 is a missing file, wherever it is reported.
	files = &fakeFiles{errs: map[string]error{
		"alice/project/.github/FUNDING.yml@main": &forges.HTTPError{StatusCode: http.StatusNotFound},
	}, files: map[string]string{"alice/project/FUNDING.yml@main": "github: alice\n"}}
	links, err := Discover(context.Background(), files, "alice", "project", "main")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if !reflect.DeepEqual(links.GitHub, []string{"alice"}) {
		t.Errorf("Discover = %+v, want the root file", links)
	}
}

func TestDiscoverRequiresAFileService(t *testing.T) {
	if _, err := Discover(context.Background(), nil, "alice", "project", "main"); err == nil {
		t.Fatal("Discover(nil) returned no error")
	}
}

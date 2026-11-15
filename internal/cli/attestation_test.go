package cli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/gitlab"
	"github.com/git-pkgs/forge/internal/config"
	"github.com/git-pkgs/forge/internal/resolve"
)

const (
	testArtifactHex    = "5db1fee4b5703808c48078a76768b155b421b210c0761cd6a5d223f4d99f1eaa"
	testArtifactDigest = "sha256:" + testArtifactHex
	testProvenanceType = "https://slsa.dev/provenance/v1"
)

type mockAttestationService struct {
	atts        []forges.Attestation
	listErr     error
	bundles     map[string]string // keyed by Attestation.ID
	downloadErr error

	listDigest string
	listOpts   forges.ListAttestationOpts
	downloads  []string
}

func (m *mockAttestationService) List(_ context.Context, _, _, digest string, opts forges.ListAttestationOpts) ([]forges.Attestation, error) {
	m.listDigest = digest
	m.listOpts = opts
	return m.atts, m.listErr
}

func (m *mockAttestationService) Download(_ context.Context, _, _ string, att forges.Attestation) (json.RawMessage, error) {
	m.downloads = append(m.downloads, att.ID)
	if m.downloadErr != nil {
		return nil, m.downloadErr
	}
	return json.RawMessage(m.bundles[att.ID]), nil
}

// runAttestationCLI runs the root command with f as the resolved forge for
// owner/repo and returns what it wrote to stdout.
func runAttestationCLI(t *testing.T, f forges.Forge, args ...string) (string, error) {
	t.Helper()
	resetCmd(rootCmd)
	t.Cleanup(func() {
		resetCmd(rootCmd)
		rootCmd.SetArgs(nil)
	})
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("FORGE_HOST", "")
	t.Setenv("FORGE_READ_ONLY", "")
	config.ResetCache()
	t.Cleanup(config.ResetCache)
	resolve.SetTestForge(f, "owner", "repo", "example.com")
	t.Cleanup(resolve.ResetTestForge)

	rootCmd.SetArgs(args)
	return captureStdout(t, rootCmd.Execute)
}

func TestAttestationSubcommands(t *testing.T) {
	want := map[string]bool{"list": false, "download": false}
	for _, cmd := range attestationCmd.Commands() {
		if _, ok := want[cmd.Name()]; ok {
			want[cmd.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

func TestArtifactDigest(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "artifact.tar.gz")
	if err := os.WriteFile(artifact, []byte("hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// sha256 of "hello\n"
	const helloDigest = "sha256:5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"

	tests := []struct {
		name    string
		arg     string
		want    string
		wantErr error
		errText string
	}{
		{name: "digest", arg: testArtifactDigest, want: testArtifactDigest},
		{name: "uppercase hex", arg: "sha256:" + strings.ToUpper(testArtifactHex), want: testArtifactDigest},
		{name: "file", arg: artifact, want: helloDigest},
		{name: "short digest", arg: "sha256:abc", wantErr: forges.ErrInvalidDigest},
		{name: "other algorithm", arg: "sha512:" + testArtifactHex + testArtifactHex, wantErr: forges.ErrInvalidDigest},
		{name: "missing file", arg: filepath.Join(dir, "missing"), errText: "reading artifact"},
		{name: "directory", arg: dir, errText: "reading artifact"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := artifactDigest(tt.arg)
			switch {
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("artifactDigest(%q) error = %v, want %v", tt.arg, err, tt.wantErr)
				}
			case tt.errText != "":
				if err == nil || !strings.Contains(err.Error(), tt.errText) {
					t.Fatalf("artifactDigest(%q) error = %v, want %q", tt.arg, err, tt.errText)
				}
			default:
				if err != nil {
					t.Fatalf("artifactDigest(%q): %v", tt.arg, err)
				}
				if got != tt.want {
					t.Errorf("artifactDigest(%q) = %q, want %q", tt.arg, got, tt.want)
				}
			}
		})
	}
}

func TestLooksLikeDigest(t *testing.T) {
	for s, want := range map[string]bool{
		"sha512:abcdef":    true,
		"sha256:abc":       true,
		"md5:ABCDEF":       true,
		`C:\artifact.zip`:  false,
		"c:artifact.zip":   false,
		"sha256:":          false,
		":abcdef":          false,
		"artifact.tar.gz":  false,
		"dir/sha256:abcde": false,
	} {
		if got := looksLikeDigest(s); got != want {
			t.Errorf("looksLikeDigest(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestAttestationListOutput(t *testing.T) {
	created := time.Date(2025, 10, 7, 20, 59, 27, 0, time.UTC)
	svc := &mockAttestationService{atts: []forges.Attestation{
		{ID: "1", SubjectDigest: testArtifactDigest, PredicateType: testProvenanceType, Status: "success", CreatedAt: created},
		{ID: "2", SubjectDigest: testArtifactDigest, PredicateType: "https://cyclonedx.org/bom", Status: "success", CreatedAt: created},
	}}
	f := &mockForge{attestationService: svc}

	stdout, err := runAttestationCLI(t, f, "attestation", "list", testArtifactDigest, "--predicate-type", "provenance", "-L", "5")
	if err != nil {
		t.Fatalf("attestation list: %v", err)
	}
	if svc.listDigest != testArtifactDigest {
		t.Errorf("List digest = %q, want %q", svc.listDigest, testArtifactDigest)
	}
	if svc.listOpts.PredicateType != "provenance" || svc.listOpts.Limit != 5 {
		t.Errorf("List opts = %+v, want predicate type provenance and limit 5", svc.listOpts)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Fatalf("table has %d lines, want header and 2 rows:\n%s", len(lines), stdout)
	}
	for _, want := range []string{"PREDICATE TYPE", testProvenanceType, "https://cyclonedx.org/bom", "success", "2025-10-07"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("table output missing %q:\n%s", want, stdout)
		}
	}

	stdout, err = runAttestationCLI(t, f, "--output", "plain", "attestation", "list", testArtifactDigest)
	if err != nil {
		t.Fatalf("attestation list --output plain: %v", err)
	}
	if want := "1\t" + testProvenanceType + "\n2\thttps://cyclonedx.org/bom\n"; stdout != want {
		t.Errorf("plain output = %q, want %q", stdout, want)
	}

	stdout, err = runAttestationCLI(t, f, "--output", "json", "attestation", "list", testArtifactDigest)
	if err != nil {
		t.Fatalf("attestation list --output json: %v", err)
	}
	var got []forges.Attestation
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decoding JSON output: %v\n%s", err, stdout)
	}
	if len(got) != 2 || got[0].ID != "1" || got[1].PredicateType != "https://cyclonedx.org/bom" {
		t.Errorf("JSON output = %+v", got)
	}
}

// dsseBundle returns a Sigstore bundle whose DSSE envelope holds an in-toto
// statement with the given predicate type.
func dsseBundle(predicateType string) string {
	statement := fmt.Sprintf(`{"_type":"https://in-toto.io/Statement/v1","predicateType":%q}`, predicateType)
	return fmt.Sprintf(`{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json","dsseEnvelope":{"payload":%q,"payloadType":"application/vnd.in-toto+json"}}`,
		base64.StdEncoding.EncodeToString([]byte(statement)))
}

func TestBundlePredicateType(t *testing.T) {
	tests := map[string]struct {
		bundle string
		want   string
	}{
		"dsse statement":    {bundle: dsseBundle(testProvenanceType), want: testProvenanceType},
		"message signature": {bundle: `{"messageSignature":{"signature":"abc"}}`},
		"invalid payload":   {bundle: `{"dsseEnvelope":{"payload":"not base64!"}}`},
		"payload not json":  {bundle: `{"dsseEnvelope":{"payload":"` + base64.StdEncoding.EncodeToString([]byte("nope")) + `"}}`},
		"not json":          {bundle: `nope`},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := bundlePredicateType(json.RawMessage(tt.bundle)); got != tt.want {
				t.Errorf("bundlePredicateType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAttestationListReadsMissingPredicateTypesFromBundles(t *testing.T) {
	svc := &mockAttestationService{
		atts: []forges.Attestation{
			{ID: "1", SubjectDigest: testArtifactDigest},
			{ID: "2", SubjectDigest: testArtifactDigest, PredicateType: "https://cyclonedx.org/bom"},
			{ID: "3", SubjectDigest: testArtifactDigest},
		},
		bundles: map[string]string{
			"1": dsseBundle(testProvenanceType),
			"3": `{"messageSignature":{"signature":"abc"}}`,
		},
	}

	stdout, err := runAttestationCLI(t, &mockForge{attestationService: svc}, "--output", "plain", "attestation", "list", testArtifactDigest)
	if err != nil {
		t.Fatalf("attestation list: %v", err)
	}
	if want := "1\t" + testProvenanceType + "\n2\thttps://cyclonedx.org/bom\n3\t\n"; stdout != want {
		t.Errorf("plain output = %q, want %q", stdout, want)
	}
	if strings.Join(svc.downloads, ",") != "1,3" {
		t.Errorf("downloaded bundles %v, want only those without a predicate type", svc.downloads)
	}

	svc = &mockAttestationService{atts: []forges.Attestation{{ID: "1"}}, downloadErr: forges.ErrNotFound}
	if _, err := runAttestationCLI(t, &mockForge{attestationService: svc}, "attestation", "list", testArtifactDigest); !errors.Is(err, forges.ErrNotFound) {
		t.Errorf("download failure error = %v, want ErrNotFound", err)
	}
}

func TestAttestationListEmptyJSONIsArray(t *testing.T) {
	f := &mockForge{attestationService: &mockAttestationService{}}

	stdout, err := runAttestationCLI(t, f, "--output", "json", "attestation", "list", testArtifactDigest)
	if err != nil {
		t.Fatalf("attestation list: %v", err)
	}
	if strings.TrimSpace(stdout) != "[]" {
		t.Errorf("JSON output = %q, want []", stdout)
	}
}

func TestAttestationListErrors(t *testing.T) {
	_, err := runAttestationCLI(t, &mockForge{attestationService: &mockAttestationService{listErr: forges.ErrNotSupported}},
		"attestation", "list", testArtifactDigest)
	if err == nil || err.Error() != "attestations is not supported by this forge" {
		t.Errorf("unsupported forge error = %v", err)
	}

	svc := &mockAttestationService{}
	_, err = runAttestationCLI(t, &mockForge{attestationService: svc}, "attestation", "list", "sha256:abc")
	if !errors.Is(err, forges.ErrInvalidDigest) {
		t.Errorf("invalid digest error = %v, want ErrInvalidDigest", err)
	}
	if svc.listDigest != "" {
		t.Errorf("List was called with %q for an invalid digest", svc.listDigest)
	}
}

func TestAttestationDownloadWritesJSONLines(t *testing.T) {
	svc := &mockAttestationService{
		atts: []forges.Attestation{{ID: "1"}, {ID: "2"}},
		bundles: map[string]string{
			"1": "{\n  \"mediaType\": \"application/vnd.dev.sigstore.bundle.v0.3+json\",\n  \"n\": 1\n}",
			"2": `{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json","n":2}`,
		},
	}
	dir := t.TempDir()
	t.Chdir(dir)
	artifact := filepath.Join(dir, "artifact.bin")
	if err := os.WriteFile(artifact, []byte("hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := artifactDigest(artifact)
	if err != nil {
		t.Fatal(err)
	}

	stdout, err := runAttestationCLI(t, &mockForge{attestationService: svc}, "attestation", "download", "artifact.bin")
	if err != nil {
		t.Fatalf("attestation download: %v", err)
	}
	if svc.listDigest != digest {
		t.Errorf("List digest = %q, want %q", svc.listDigest, digest)
	}
	outPath := attestationFileName(digest)
	if stdout != "Downloaded 2 attestations to "+outPath+"\n" {
		t.Errorf("stdout = %q", stdout)
	}
	data, err := os.ReadFile(filepath.Join(dir, outPath))
	if err != nil {
		t.Fatalf("reading bundle file: %v", err)
	}
	want := `{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json","n":1}` + "\n" +
		`{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json","n":2}` + "\n"
	if string(data) != want {
		t.Errorf("bundle file = %q, want %q", data, want)
	}
}

func TestAttestationDownloadOutputFile(t *testing.T) {
	svc := &mockAttestationService{
		atts:    []forges.Attestation{{ID: "7"}},
		bundles: map[string]string{"7": `{"n":7}`},
	}
	outPath := filepath.Join(t.TempDir(), "bundles.jsonl")

	stdout, err := runAttestationCLI(t, &mockForge{attestationService: svc},
		"attestation", "download", testArtifactDigest, "-O", outPath, "--predicate-type", "sbom")
	if err != nil {
		t.Fatalf("attestation download: %v", err)
	}
	if svc.listOpts.PredicateType != "sbom" {
		t.Errorf("List predicate type = %q, want sbom", svc.listOpts.PredicateType)
	}
	if stdout != "Downloaded 1 attestation to "+outPath+"\n" {
		t.Errorf("stdout = %q", stdout)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading bundle file: %v", err)
	}
	if string(data) != "{\"n\":7}\n" {
		t.Errorf("bundle file = %q", data)
	}
}

func TestAttestationDownloadFailuresWriteNoFile(t *testing.T) {
	tests := []struct {
		name    string
		svc     *mockAttestationService
		errText string
	}{
		{
			name:    "no attestations",
			svc:     &mockAttestationService{},
			errText: "no attestations found for " + testArtifactDigest + " in owner/repo",
		},
		{
			name:    "unsupported",
			svc:     &mockAttestationService{listErr: forges.ErrNotSupported},
			errText: "attestations is not supported by this forge",
		},
		{
			name: "second download fails",
			svc: &mockAttestationService{
				atts:    []forges.Attestation{{ID: "1"}, {ID: "2"}},
				bundles: map[string]string{"1": `{"n":1}`},
			},
			errText: "invalid attestation bundle",
		},
		{
			name: "download error",
			svc: &mockAttestationService{
				atts:        []forges.Attestation{{ID: "1"}},
				downloadErr: forges.ErrNotFound,
			},
			errText: forges.ErrNotFound.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outPath := filepath.Join(t.TempDir(), "bundles.jsonl")
			_, err := runAttestationCLI(t, &mockForge{attestationService: tt.svc},
				"attestation", "download", testArtifactDigest, "-O", outPath)
			if err == nil || !strings.Contains(err.Error(), tt.errText) {
				t.Fatalf("error = %v, want %q", err, tt.errText)
			}
			if _, err := os.Stat(outPath); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("bundle file exists after failure (stat error %v)", err)
			}
		})
	}
}

// TestAttestationCommandsAgainstGitLab runs list and download through the
// real GitLab backend against a fake attestation API.
func TestAttestationCommandsAgainstGitLab(t *testing.T) {
	const bundle = `{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json","dsseEnvelope":{"payloadType":"application/vnd.in-toto+json"}}`
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/owner%2Frepo/attestations/"+testArtifactHex, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `[{"id":103,"iid":3,"project_id":1,"status":"success",
			"created_at":"2025-10-07T20:59:27.085Z","expire_at":"2027-10-07T20:59:26.967Z",
			"predicate_kind":"provenance","predicate_type":%q,"subject_digest":%q,
			"download_url":"https://gitlab.example.com/api/v4/projects/1/attestations/3/download"}]`,
			testProvenanceType, testArtifactHex)
	})
	mux.HandleFunc("GET /api/v4/projects/owner%2Frepo/attestations/3/download", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, bundle)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	f := gitlab.New(srv.URL, "", srv.Client())

	stdout, err := runAttestationCLI(t, f, "--output", "plain", "attestation", "list", testArtifactDigest)
	if err != nil {
		t.Fatalf("attestation list: %v", err)
	}
	if want := "3\t" + testProvenanceType + "\n"; stdout != want {
		t.Errorf("list output = %q, want %q", stdout, want)
	}

	outPath := filepath.Join(t.TempDir(), "bundles.jsonl")
	if _, err := runAttestationCLI(t, f, "attestation", "download", testArtifactDigest, "-O", outPath); err != nil {
		t.Fatalf("attestation download: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading bundle file: %v", err)
	}
	if string(data) != bundle+"\n" {
		t.Errorf("bundle file = %q, want %q", data, bundle+"\n")
	}
}

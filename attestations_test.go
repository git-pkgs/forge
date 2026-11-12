package forges

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const testDigestHex = "5db1fee4b5703808c48078a76768b155b421b210c0761cd6a5d223f4d99f1eaa"

func TestParseSHA256Digest(t *testing.T) {
	tests := []struct {
		name    string
		digest  string
		want    string
		wantErr bool
	}{
		{"lowercase", "sha256:" + testDigestHex, testDigestHex, false},
		{"uppercase hex", "sha256:" + strings.ToUpper(testDigestHex), testDigestHex, false},
		{"bare hex", testDigestHex, "", true},
		{"other algorithm", "sha512:" + testDigestHex, "", true},
		{"uppercase algorithm", "SHA256:" + testDigestHex, "", true},
		{"too short", "sha256:" + testDigestHex[:63], "", true},
		{"too long", "sha256:" + testDigestHex + "a", "", true},
		{"non-hex", "sha256:" + testDigestHex[:63] + "g", "", true},
		{"path separator", "sha256:" + testDigestHex[:62] + "/a", "", true},
		{"empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSHA256Digest(tt.digest)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidDigest) {
					t.Errorf("ParseSHA256Digest(%q) error = %v, want ErrInvalidDigest", tt.digest, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSHA256Digest(%q) error = %v", tt.digest, err)
			}
			if got != tt.want {
				t.Errorf("ParseSHA256Digest(%q) = %q, want %q", tt.digest, got, tt.want)
			}
		})
	}
}

func TestClientListAttestationsRoutes(t *testing.T) {
	svc := &mockAttestationService{attestations: []Attestation{{ID: "1"}, {ID: "2"}}}
	c := &Client{
		forges: map[string]Forge{"example.com": &mockForge{attestations: svc}},
		tokens: make(map[string]string),
	}

	opts := ListAttestationOpts{PredicateType: "provenance", Limit: 5}
	atts, err := c.ListAttestations(context.Background(), "https://example.com/test/repo", "sha256:"+testDigestHex, opts)
	if err != nil {
		t.Fatalf("ListAttestations() error = %v", err)
	}
	if len(atts) != 2 {
		t.Fatalf("expected 2 attestations, got %d", len(atts))
	}
	if svc.lastOwner != "test" || svc.lastRepo != "repo" || svc.lastDigest != "sha256:"+testDigestHex {
		t.Errorf("backend received owner=%q repo=%q digest=%q", svc.lastOwner, svc.lastRepo, svc.lastDigest)
	}
	if svc.lastOpts != opts {
		t.Errorf("backend received opts %+v, want %+v", svc.lastOpts, opts)
	}
}

func TestClientDownloadAttestationRoutes(t *testing.T) {
	svc := &mockAttestationService{bundle: []byte(`{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json"}`)}
	c := &Client{
		forges: map[string]Forge{"example.com": &mockForge{attestations: svc}},
		tokens: make(map[string]string),
	}

	bundle, err := c.DownloadAttestation(context.Background(), "https://example.com/test/repo", Attestation{ID: "7"})
	if err != nil {
		t.Fatalf("DownloadAttestation() error = %v", err)
	}
	if string(bundle) != string(svc.bundle) {
		t.Errorf("DownloadAttestation() = %s, want %s", bundle, svc.bundle)
	}
	if svc.lastOwner != "test" || svc.lastRepo != "repo" || svc.lastAtt.ID != "7" {
		t.Errorf("backend received owner=%q repo=%q id=%q", svc.lastOwner, svc.lastRepo, svc.lastAtt.ID)
	}
}

func TestClientListAttestationsUnregisteredDomain(t *testing.T) {
	c := NewClient()
	if _, err := c.ListAttestations(context.Background(), "https://example.com/test/repo", "sha256:"+testDigestHex, ListAttestationOpts{}); err == nil {
		t.Error("ListAttestations() error = nil, want error for unregistered domain")
	}
}

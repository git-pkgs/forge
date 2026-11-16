package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"
	"strings"

	"github.com/git-pkgs/forge"
	"github.com/git-pkgs/forge/internal/output"
	"github.com/git-pkgs/forge/internal/resolve"
	"github.com/spf13/cobra"
)

const attestationDigestPrefix = "sha256:"

var attestationCmd = &cobra.Command{
	Use:   "attestation",
	Short: "Find and download artifact attestations",
	Long: `Find and download the artifact attestations, such as SLSA provenance and
SBOMs signed as Sigstore bundles, that a repository holds for an artifact.

Each command takes either a path to the artifact, which is hashed locally with
SHA-256, or its digest in the form sha256:<hex>. Bundles are fetched as the
forge stores them and are not verified.`,
}

func init() {
	rootCmd.AddCommand(attestationCmd)
	attestationCmd.AddCommand(attestationListCmd())
	attestationCmd.AddCommand(attestationDownloadCmd())
}

func attestationListCmd() *cobra.Command {
	var (
		flagPredicateType string
		flagLimit         int
	)

	cmd := &cobra.Command{
		Use:   "list <file|sha256:digest>",
		Short: "List attestations for an artifact",
		Long:  "Lists the attestations the repository holds for an artifact. When the forge doesn't report an attestation's predicate type, as GitHub doesn't, its bundle is downloaded to read the type from the signed statement.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			digest, err := artifactDigest(args[0])
			if err != nil {
				return err
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			atts, err := forge.Attestations().List(cmd.Context(), owner, repoName, digest, forges.ListAttestationOpts{
				PredicateType: flagPredicateType,
				Limit:         flagLimit,
			})
			if err != nil {
				return notSupported(err, "attestations")
			}
			if err := fillPredicateTypes(cmd.Context(), forge.Attestations(), owner, repoName, atts); err != nil {
				return err
			}

			p := printer()
			if p.Format == output.JSON {
				if atts == nil {
					atts = []forges.Attestation{}
				}
				return p.PrintJSON(atts)
			}

			if p.Format == output.Plain {
				lines := make([]string, len(atts))
				for i, a := range atts {
					lines[i] = strings.Join([]string{
						output.Sanitize(a.ID),
						output.Sanitize(a.PredicateType),
						output.Sanitize(a.Status),
						attestationCreated(a),
					}, "\t")
				}
				p.PrintPlain(lines)
				return nil
			}

			headers := []string{"ID", "PREDICATE TYPE", "STATUS", "CREATED"}
			rows := make([][]string, len(atts))
			for i, a := range atts {
				rows[i] = []string{
					output.Sanitize(a.ID),
					output.Sanitize(a.PredicateType),
					output.Sanitize(a.Status),
					attestationCreated(a),
				}
			}
			p.PrintTable(headers, rows)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagPredicateType, "predicate-type", "", "Only include this predicate type (a URI, or provenance or sbom)")
	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 0, "Maximum number of attestations")
	return cmd
}

func attestationCreated(a forges.Attestation) string {
	if a.CreatedAt.IsZero() {
		return ""
	}
	return a.CreatedAt.Format("2006-01-02")
}

func attestationDownloadCmd() *cobra.Command {
	var (
		flagPredicateType string
		flagLimit         int
		flagOutput        string
	)

	cmd := &cobra.Command{
		Use:   "download <file|sha256:digest>",
		Short: "Download attestation bundles for an artifact",
		Long: `Downloads every attestation bundle the repository holds for an artifact and
writes them as JSON Lines, one Sigstore bundle per line, to a file named after
the artifact digest (sha256:<hex>.jsonl, or sha256-<hex>.jsonl on Windows).
The file can be checked with gh attestation verify --bundle. Nothing is
written if any bundle fails to download.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			digest, err := artifactDigest(args[0])
			if err != nil {
				return err
			}

			forge, owner, repoName, _, err := resolve.Repo(flagRepo, flagForgeType)
			if err != nil {
				return err
			}

			atts, err := forge.Attestations().List(cmd.Context(), owner, repoName, digest, forges.ListAttestationOpts{
				PredicateType: flagPredicateType,
				Limit:         flagLimit,
			})
			if err != nil {
				return notSupported(err, "attestations")
			}
			if len(atts) == 0 {
				return fmt.Errorf("no attestations found for %s in %s", digest, repoDisplayName(owner, repoName))
			}

			// Fetch every bundle before writing so a failed download
			// doesn't leave a partial file behind.
			var buf bytes.Buffer
			for _, att := range atts {
				bundle, err := forge.Attestations().Download(cmd.Context(), owner, repoName, att)
				if err != nil {
					return notSupported(err, "attestations")
				}
				if err := json.Compact(&buf, bundle); err != nil {
					return fmt.Errorf("invalid attestation bundle: %w", err)
				}
				buf.WriteByte('\n')
			}

			outPath := flagOutput
			if outPath == "" {
				outPath = attestationFileName(digest)
			}
			if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
				return fmt.Errorf("writing file: %w", err)
			}

			noun := "attestations"
			if len(atts) == 1 {
				noun = "attestation"
			}
			_, _ = fmt.Fprintf(os.Stdout, "Downloaded %d %s to %s\n", len(atts), noun, outPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagPredicateType, "predicate-type", "", "Only include this predicate type (a URI, or provenance or sbom)")
	cmd.Flags().IntVarP(&flagLimit, "limit", "L", 0, "Maximum number of attestations")
	cmd.Flags().StringVarP(&flagOutput, "output-file", "O", "", "Output file path (defaults to the artifact digest with a .jsonl extension)")
	return cmd
}

// fillPredicateTypes sets the predicate type of each attestation listed
// without one from the in-toto statement in its bundle.
func fillPredicateTypes(ctx context.Context, svc forges.AttestationService, owner, repo string, atts []forges.Attestation) error {
	for i := range atts {
		if atts[i].PredicateType != "" {
			continue
		}
		bundle, err := svc.Download(ctx, owner, repo, atts[i])
		if err != nil {
			return err
		}
		atts[i].PredicateType = bundlePredicateType(bundle)
	}
	return nil
}

// bundlePredicateType returns the predicateType of the in-toto statement in a
// Sigstore bundle's DSSE envelope, or "" if the bundle doesn't hold one.
func bundlePredicateType(bundle json.RawMessage) string {
	var b struct {
		DSSEEnvelope struct {
			Payload []byte `json:"payload"` // base64 in the bundle
		} `json:"dsseEnvelope"`
	}
	if err := json.Unmarshal(bundle, &b); err != nil {
		return ""
	}
	var statement struct {
		PredicateType string `json:"predicateType"`
	}
	if err := json.Unmarshal(b.DSSEEnvelope.Payload, &statement); err != nil {
		return ""
	}
	return statement.PredicateType
}

// artifactDigest returns the sha256:<hex> digest for arg, which is either a
// digest in that form or the path of an artifact to hash.
func artifactDigest(arg string) (string, error) {
	if strings.HasPrefix(arg, attestationDigestPrefix) {
		hexDigest, err := forges.ParseSHA256Digest(arg)
		if err != nil {
			return "", err
		}
		return attestationDigestPrefix + hexDigest, nil
	}

	f, err := os.Open(arg)
	if err != nil {
		// Something like sha512:<hex> is a digest forge can't look up rather
		// than a missing file.
		if errors.Is(err, fs.ErrNotExist) && looksLikeDigest(arg) {
			_, digestErr := forges.ParseSHA256Digest(arg)
			return "", digestErr
		}
		return "", fmt.Errorf("reading artifact: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("reading artifact: %w", err)
	}
	return attestationDigestPrefix + hex.EncodeToString(h.Sum(nil)), nil
}

// looksLikeDigest reports whether s has the form <algorithm>:<hex>.
func looksLikeDigest(s string) bool {
	alg, value, ok := strings.Cut(s, ":")
	if !ok || alg == "" || value == "" {
		return false
	}
	for _, r := range alg {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	_, err := hex.DecodeString(value)
	return err == nil || errors.Is(err, hex.ErrLength)
}

// attestationFileName names the bundle file after the artifact digest, the
// same way gh attestation download does. Windows doesn't allow ':' in file
// names, so it is replaced with '-' there.
func attestationFileName(digest string) string {
	if runtime.GOOS == "windows" {
		digest = strings.ReplaceAll(digest, ":", "-")
	}
	return digest + ".jsonl"
}

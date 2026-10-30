// Package funding discovers the funding links a repository advertises, the same
// way GitHub does for the Sponsor button: a FUNDING.yml file, looked up in the
// places GitHub looks.
package funding

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	forges "github.com/git-pkgs/forge"
)

// Links is the parsed content of a FUNDING.yml file.
//
// The field names follow the file's keys, not GitHub's button labels. A file
// may declare several destinations, and each one is a handle the platform
// understands, except Custom, which holds URLs.
type Links struct {
	GitHub          []string `json:"github,omitempty"`
	Patreon         string   `json:"patreon,omitempty"`
	OpenCollective  string   `json:"open_collective,omitempty"`
	KoFi            string   `json:"ko_fi,omitempty"`
	Tidelift        string   `json:"tidelift,omitempty"`
	CommunityBridge string   `json:"community_bridge,omitempty"`
	Liberapay       string   `json:"liberapay,omitempty"`
	IssueHunt       string   `json:"issuehunt,omitempty"`
	LFXCrowdfunding string   `json:"lfx_crowdfunding,omitempty"`
	Polar           string   `json:"polar,omitempty"`
	BuyMeACoffee    string   `json:"buy_me_a_coffee,omitempty"`
	ThanksDev       string   `json:"thanks_dev,omitempty"`
	Otechie         string   `json:"otechie,omitempty"`
	Custom          []string `json:"custom,omitempty"`
}

// Empty reports whether the file declared no funding destination.
func (l Links) Empty() bool {
	return len(l.GitHub) == 0 && len(l.Custom) == 0 &&
		l.Patreon == "" && l.OpenCollective == "" && l.KoFi == "" &&
		l.Tidelift == "" && l.CommunityBridge == "" && l.Liberapay == "" &&
		l.IssueHunt == "" && l.LFXCrowdfunding == "" && l.Polar == "" &&
		l.BuyMeACoffee == "" && l.ThanksDev == "" && l.Otechie == ""
}

// URL returns the funding page GitHub links for a handle, or an empty string
// for Custom: those entries are already URLs and are returned unchanged.
func (l Links) URL(platform, handle string) string {
	switch platform {
	case "github":
		return "https://github.com/sponsors/" + handle
	case "patreon":
		return "https://www.patreon.com/" + handle
	case "open_collective":
		return "https://opencollective.com/" + handle
	case "ko_fi":
		return "https://ko-fi.com/" + handle
	case "tidelift":
		return "https://tidelift.com/funding/github/" + handle
	case "community_bridge":
		return "https://funding.communitybridge.org/projects/" + handle
	case "liberapay":
		return "https://liberapay.com/" + handle
	case "issuehunt":
		return "https://issuehunt.io/r/" + handle
	case "lfx_crowdfunding":
		return "https://crowdfunding.lfx.linuxfoundation.org/projects/" + handle
	case "polar":
		return "https://polar.sh/" + handle
	case "buy_me_a_coffee":
		return "https://www.buymeacoffee.com/" + handle
	case "thanks_dev":
		return "https://thanks.dev/" + handle
	case "otechie":
		return "https://otechie.com/" + handle
	}
	return ""
}

// locations is where GitHub looks for a funding file, in order: the .github
// directory, the repository root, docs/, and finally the owner's .github
// repository. Discovery stops at the first file it finds, even when that file
// declares nothing: existence, not content, decides which file GitHub reads.
var locations = []Location{
	{Path: ".github/FUNDING.yml"},
	{Path: "FUNDING.yml"},
	{Path: "docs/FUNDING.yml"},
	{Repo: ".github", Path: ".github/FUNDING.yml"},
	{Repo: ".github", Path: "FUNDING.yml"},
}

// Location is one file a funding file may live in. An empty Repo means the
// repository being asked about; a set one names another repository of the same
// owner, which is how the owner's .github repository is described.
type Location struct {
	Repo string
	Path string
}

// Discover reads the funding file GitHub would read for owner/repo. ref is the
// branch or commit to read from; empty means the repository's default branch.
// The owner's .github repository is always read from its own default branch,
// because a tag or branch name from owner/repo does not exist there.
//
// A file that is not there is not an error: Discover moves on to the next
// location and returns empty Links when none of them has one. A forge that
// cannot read files at all (ErrNotSupported, which the Tangled and Bitbucket
// backends return for every Get) is treated the same way. Any other error — an
// authentication failure, a rate limit — is returned, wrapped, because it says
// nothing about whether the repository is funded; errors.Is and errors.As see
// through the wrap.
func Discover(ctx context.Context, files forges.FileService, owner, repo, ref string) (Links, error) {
	if files == nil {
		return Links{}, errors.New("funding: nil file service")
	}
	for _, loc := range locations {
		fileRepo, probeRef := repo, ref
		if loc.Repo != "" {
			fileRepo, probeRef = loc.Repo, ""
		}
		content, err := files.Get(ctx, owner, fileRepo, loc.Path, probeRef)
		if err != nil {
			if isMissing(err) {
				continue
			}
			return Links{}, fmt.Errorf("funding: read %s/%s %s: %w", owner, fileRepo, loc.Path, err)
		}
		if content == nil {
			continue
		}
		links, err := Parse(content.Content)
		if err != nil {
			return Links{}, fmt.Errorf("funding: parse %s/%s %s: %w", owner, fileRepo, loc.Path, err)
		}
		return links, nil
	}
	return Links{}, nil
}

// isMissing reports whether an error means there is nothing to read: the file
// is not there, or the forge has no file API at all.
func isMissing(err error) bool {
	if errors.Is(err, forges.ErrNotFound) || errors.Is(err, forges.ErrNotSupported) {
		return true
	}
	var httpErr *forges.HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound
}

// scalarKeys are the FUNDING.yml keys that hold a single handle.
var scalarKeys = map[string]func(*Links) *string{
	"patreon":          func(l *Links) *string { return &l.Patreon },
	"open_collective":  func(l *Links) *string { return &l.OpenCollective },
	"ko_fi":            func(l *Links) *string { return &l.KoFi },
	"tidelift":         func(l *Links) *string { return &l.Tidelift },
	"community_bridge": func(l *Links) *string { return &l.CommunityBridge },
	"liberapay":        func(l *Links) *string { return &l.Liberapay },
	"issuehunt":        func(l *Links) *string { return &l.IssueHunt },
	"lfx_crowdfunding": func(l *Links) *string { return &l.LFXCrowdfunding },
	"polar":            func(l *Links) *string { return &l.Polar },
	"buy_me_a_coffee":  func(l *Links) *string { return &l.BuyMeACoffee },
	"thanks_dev":       func(l *Links) *string { return &l.ThanksDev },
	"otechie":          func(l *Links) *string { return &l.Otechie },
}

// listKeys are the FUNDING.yml keys that hold a list of handles or URLs.
var listKeys = map[string]func(*Links) *[]string{
	"github": func(l *Links) *[]string { return &l.GitHub },
	"custom": func(l *Links) *[]string { return &l.Custom },
}

// Parse reads the body of a FUNDING.yml file.
//
// The format is the small flat subset GitHub accepts: one "key: value" per line,
// a value in quotes, a bare handle, an inline list "[a, b]", or a list of "- x"
// lines below the key. Comments start at "#", a "---" document marker is
// ignored, and a leading UTF-8 BOM is stripped. Unknown keys are ignored, so a
// file that uses a platform added after this package was written still parses,
// and a line that is none of those shapes is an error rather than a silent skip.
//
// Two shapes are outside this subset rather than mis-parsed into something else:
// a "#" preceded by a space inside a quoted value, and a comma inside a quoted
// inline-list item. No FUNDING.yml seen in the wild uses either.
func Parse(content []byte) (Links, error) {
	var links Links
	var pending *[]string // the list a "key:" line without a value opens

	content = bytes.TrimPrefix(content, []byte("\ufeff"))

	flush := func() {
		pending = nil
	}

	for number, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimRight(raw, " \t\r")
		line = stripComment(line)
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || trimmed == "---" || trimmed == "..." {
			continue
		}

		if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
			if pending == nil {
				return Links{}, fmt.Errorf("line %d: list item outside a list key", number+1)
			}
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			if item == "" {
				return Links{}, fmt.Errorf("line %d: empty list item", number+1)
			}
			*pending = append(*pending, unquote(item))
			continue
		}

		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			return Links{}, fmt.Errorf("line %d: want \"key: value\", got %q", number+1, trimmed)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		flush()

		if setter, known := scalarKeys[key]; known {
			if value == "" {
				continue
			}
			*setter(&links) = unquote(value)
			continue
		}
		if setter, known := listKeys[key]; known {
			list := setter(&links)
			switch {
			case value == "":
				pending = list
			case strings.HasPrefix(value, "["):
				items, err := parseInlineList(value)
				if err != nil {
					return Links{}, fmt.Errorf("line %d: %w", number+1, err)
				}
				*list = append(*list, items...)
			default:
				*list = append(*list, unquote(value))
			}
			continue
		}
		// An unknown key: a platform this package does not know about yet.
	}
	return links, nil
}

// stripComment removes a "#" comment. A "#" only starts a comment at the start
// of the line or after whitespace, so a URL fragment survives.
func stripComment(line string) string {
	for i := 0; i < len(line); i++ {
		if line[i] != '#' {
			continue
		}
		if i == 0 || line[i-1] == ' ' || line[i-1] == '\t' {
			return line[:i]
		}
	}
	return line
}

// parseInlineList reads "[a, b, 'c']".
func parseInlineList(value string) ([]string, error) {
	if !strings.HasSuffix(value, "]") {
		return nil, fmt.Errorf("unterminated list %q", value)
	}
	inner := strings.TrimSpace(value[1 : len(value)-1])
	if inner == "" {
		return nil, nil
	}
	parts := strings.Split(inner, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := unquote(strings.TrimSpace(part))
		if item == "" {
			return nil, fmt.Errorf("empty item in list %q", value)
		}
		items = append(items, item)
	}
	return items, nil
}

// unquote removes one pair of matching quotes.
func unquote(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

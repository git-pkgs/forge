// Package funding discovers the funding links a repository advertises, the same
// way GitHub does for the Sponsor button: a FUNDING.yml file, looked up in the
// places GitHub looks.
package funding

import (
	"context"
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"

	forges "github.com/git-pkgs/forge"
)

// Links is the parsed content of a FUNDING.yml file.
type Links struct {
	GitHub          stringOrList `yaml:"github" json:"github,omitempty"`
	Patreon         string       `yaml:"patreon" json:"patreon,omitempty"`
	OpenCollective  string       `yaml:"open_collective" json:"open_collective,omitempty"`
	KoFi            string       `yaml:"ko_fi" json:"ko_fi,omitempty"`
	Tidelift        string       `yaml:"tidelift" json:"tidelift,omitempty"`
	CommunityBridge string       `yaml:"community_bridge" json:"community_bridge,omitempty"`
	Liberapay       string       `yaml:"liberapay" json:"liberapay,omitempty"`
	IssueHunt       string       `yaml:"issuehunt" json:"issuehunt,omitempty"`
	LFXCrowdfunding string       `yaml:"lfx_crowdfunding" json:"lfx_crowdfunding,omitempty"`
	Polar           string       `yaml:"polar" json:"polar,omitempty"`
	BuyMeACoffee    string       `yaml:"buy_me_a_coffee" json:"buy_me_a_coffee,omitempty"`
	ThanksDev       string       `yaml:"thanks_dev" json:"thanks_dev,omitempty"`
	Otechie         string       `yaml:"otechie" json:"otechie,omitempty"`
	Custom          stringOrList `yaml:"custom" json:"custom,omitempty"`
}

// Empty reports whether the file declared no funding destination.
func (l Links) Empty() bool {
	return len(l.GitHub) == 0 && len(l.Custom) == 0 &&
		l.Patreon == "" && l.OpenCollective == "" && l.KoFi == "" &&
		l.Tidelift == "" && l.CommunityBridge == "" && l.Liberapay == "" &&
		l.IssueHunt == "" && l.LFXCrowdfunding == "" && l.Polar == "" &&
		l.BuyMeACoffee == "" && l.ThanksDev == "" && l.Otechie == ""
}

// URL returns the page GitHub links for a handle on a platform, or an empty
// string for a platform it does not know. Custom entries are already URLs.
func URL(platform, handle string) string {
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

// locations is where GitHub looks for a funding file, in order.
var locations = []location{
	{path: ".github/FUNDING.yml"},
	{path: "FUNDING.yml"},
	{path: "docs/FUNDING.yml"},
	{repo: ".github", path: ".github/FUNDING.yml"},
	{repo: ".github", path: "FUNDING.yml"},
}

// location is one file a funding file may live in. A set repo names another
// repository of the same owner, read from its own default branch.
type location struct {
	repo string
	path string
}

// Discover reads the funding file GitHub would read for owner/repo; an empty
// ref reads the default branch, and a missing file moves to the next location.
func Discover(ctx context.Context, files forges.FileService, owner, repo, ref string) (Links, error) {
	if files == nil {
		return Links{}, errors.New("funding: nil file service")
	}
	for _, loc := range locations {
		fileRepo, probeRef := repo, ref
		if loc.repo != "" {
			fileRepo, probeRef = loc.repo, ""
		}
		content, err := files.Get(ctx, owner, fileRepo, loc.path, probeRef)
		if err != nil {
			if isMissing(err) {
				continue
			}
			return Links{}, fmt.Errorf("funding: read %s/%s %s: %w", owner, fileRepo, loc.path, err)
		}
		if content == nil {
			continue
		}
		links, err := Parse(content.Content)
		if err != nil {
			return Links{}, fmt.Errorf("funding: parse %s/%s %s: %w", owner, fileRepo, loc.path, err)
		}
		return links, nil
	}
	return Links{}, nil
}

// isMissing reports whether an error means there is nothing to read: the file
// is not there, or the forge has no file API at all.
func isMissing(err error) bool {
	return errors.Is(err, forges.ErrNotFound) || errors.Is(err, forges.ErrNotSupported)
}

// Parse reads the body of a FUNDING.yml file. Keys this package does not know
// are ignored, so a platform added later still parses.
func Parse(content []byte) (Links, error) {
	var links Links
	if err := yaml.Unmarshal(content, &links); err != nil {
		return Links{}, err
	}
	return links, nil
}

// stringOrList is a value GitHub accepts either as one entry or as a list of
// them: "github" and "custom" both allow either.
type stringOrList []string

func (l *stringOrList) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var value string
		if err := node.Decode(&value); err != nil {
			return err
		}
		if value != "" {
			*l = stringOrList{value}
		}
		return nil
	case yaml.SequenceNode:
		var values []string
		if err := node.Decode(&values); err != nil {
			return err
		}
		*l = values
		return nil
	}
	return fmt.Errorf("want a string or a list, got %s", node.Tag)
}

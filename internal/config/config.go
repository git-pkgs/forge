package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	dirPermissions  = 0700
	filePermissions = 0600
	goosWindows     = "windows"
)

type Config struct {
	Default DefaultSection
	Domains map[string]DomainSection
}

type DefaultSection struct {
	Output      string // table, json, plain
	ForgeType   string // default forge type
	GitProtocol string // https or ssh
}

type DomainSection struct {
	Type        string // github, gitlab, gitea, forgejo, bitbucket
	Token       string // resolved token value; only from user config, never .forge
	TokenExec   string // non-empty when token is retrieved via a shell command (from "token-cmd" config key)
	SSHHost     string // alternate host for git-over-ssh; the section name remains the API host
	GitProtocol string // https or ssh; overrides default
}

// ResolveToken returns the token for this domain. If TokenExec is set, it
// executes the command and returns its output; otherwise it returns Token.
func (ds DomainSection) ResolveToken(domain string) (string, error) {
	if ds.TokenExec != "" {
		return execValue(ds.TokenExec, domain)
	}
	return ds.Token, nil
}

// DomainForSSHHost returns the API domain (the section name) whose ssh_host
// matches the given host, or "" if none. Self-hosted GitLab in particular can
// serve git-over-ssh on a different host than the web/API, so a remote URL like
// git@ssh.gitlab.test:owner/repo needs mapping back to gitlab.test before we
// build an API client.
func (c *Config) DomainForSSHHost(sshHost string) string {
	if c == nil {
		return ""
	}
	for name, ds := range c.Domains {
		if ds.SSHHost == sshHost {
			return name
		}
	}
	return ""
}

var (
	cached   *Config
	cacheErr error
	once     sync.Once
)

// Load returns the merged config from both user and project config files.
// The result is cached so the files are parsed at most once per invocation.
func Load() (*Config, error) {
	once.Do(func() {
		cached, cacheErr = load()
	})
	return cached, cacheErr
}

// GitProtocolFor returns the configured git protocol ("https" or "ssh") for the given domain.
// Falls back to the default if not set for the domain. Returns "https" if not configured.
func GitProtocolFor(domain string) string {
	cfg, err := Load()
	if err != nil || cfg == nil {
		return "https"
	}
	if ds, ok := cfg.Domains[domain]; ok && ds.GitProtocol != "" {
		return ds.GitProtocol
	}
	if cfg.Default.GitProtocol != "" {
		return cfg.Default.GitProtocol
	}
	return "https"
}

func parseGitProtocol(v string) (string, error) {
	switch strings.ToLower(v) {
	case "ssh":
		return "ssh", nil
	case "https":
		return "https", nil
	default:
		return "", fmt.Errorf("invalid git_protocol %q: must be \"https\" or \"ssh\"", v)
	}
}

// execValue runs cmd via sh -c and returns its trimmed stdout.
// Shell features (pipes, quotes, substitutions) are supported.
// FORGE_DOMAIN is set to domain in the command environment.
// Stdin and stderr are wired to the terminal so interactive prompts
// (e.g. pinentry, rbw unlock) work and error output is visible directly.
func execValue(cmd, domain string) (string, error) {
	if runtime.GOOS == goosWindows {
		return "", fmt.Errorf("token-cmd is not supported on Windows")
	}
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", fmt.Errorf("empty command")
	}
	var stdout strings.Builder
	c := exec.Command("sh", "-c", cmd)
	c.Env = append(os.Environ(), "FORGE_DOMAIN="+domain)
	c.Stdin = os.Stdin
	c.Stdout = &stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("%q: %w", cmd, err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// ResetCache clears the cached config. Only useful in tests.
func ResetCache() {
	once = sync.Once{}
	cached = nil
	cacheErr = nil
}

func load() (*Config, error) {
	cfg := &Config{
		Domains: make(map[string]DomainSection),
	}

	userPath := UserConfigPath()
	if userPath != "" {
		if err := loadFile(cfg, userPath, true); err != nil {
			return nil, fmt.Errorf("reading user config: %w", err)
		}
	}

	projectPath := ProjectConfigPath()
	if projectPath != "" {
		if err := loadFile(cfg, projectPath, false); err != nil {
			return nil, fmt.Errorf("reading project config: %w", err)
		}
	}

	return cfg, nil
}

func loadFile(cfg *Config, path string, allowTokens bool) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	sections, err := parseINI(f)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	if def, ok := sections["default"]; ok {
		if v, ok := def["output"]; ok {
			cfg.Default.Output = v
		}
		if v, ok := def["forge-type"]; ok {
			cfg.Default.ForgeType = v
		}
		if v, ok := def["git_protocol"]; ok {
			p, err := parseGitProtocol(v)
			if err != nil {
				return fmt.Errorf("%s: [default] %w", path, err)
			}
			cfg.Default.GitProtocol = p
		}
	}

	for name, kv := range sections {
		if name == "default" {
			continue
		}
		ds := cfg.Domains[name]
		if v, ok := kv["type"]; ok {
			ds.Type = v
		}
		if v, ok := kv["git_protocol"]; ok {
			p, err := parseGitProtocol(v)
			if err != nil {
				return fmt.Errorf("%s: [%s] %w", path, name, err)
			}
			ds.GitProtocol = p
		}
		if allowTokens {
			if v, ok := kv["ssh_host"]; ok {
				ds.SSHHost = v
			}
		}
		if allowTokens {
			_, hasToken := kv["token"]
			_, hasTokenCmd := kv["token-cmd"]
			if hasToken && hasTokenCmd {
				return fmt.Errorf("%s: [%s] token and token-cmd are mutually exclusive", path, name)
			}
			if hasToken {
				ds.Token = kv["token"]
			}
			if hasTokenCmd {
				ds.TokenExec = kv["token-cmd"]
			}
		}
		cfg.Domains[name] = ds
	}

	return nil
}

// parseINI parses a simple INI file into section -> key -> value.
// Supports # and ; comments, blank lines, and [section] headers.
func parseINI(r io.Reader) (map[string]map[string]string, error) {
	sections := make(map[string]map[string]string)
	current := ""

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}

		if line[0] == '[' && line[len(line)-1] == ']' {
			current = line[1 : len(line)-1]
			if _, ok := sections[current]; !ok {
				sections[current] = make(map[string]string)
			}
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if current == "" {
			current = "default"
			if _, ok := sections[current]; !ok {
				sections[current] = make(map[string]string)
			}
		}
		sections[current][key] = value
	}
	return sections, scanner.Err()
}

// UserConfigPath returns the path to the user-level config file.
// It respects XDG_CONFIG_HOME, falling back to ~/.config/forge/config.
func UserConfigPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "forge", "config")
}

// ProjectConfigPath walks up from the current directory looking for a .forge file.
// Returns empty string if none is found.
func ProjectConfigPath() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return findProjectConfig(dir)
}

func findProjectConfig(dir string) string {
	for {
		path := filepath.Join(dir, ".forge")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// SetDomain updates or adds a domain section in the user config file.
// Creates the config directory if needed. Sets file permissions to 0600
// since the file may contain tokens.
func SetDomain(domain, token, tokenCmd, forgeType string) error {
	path := UserConfigPath()
	if path == "" {
		return fmt.Errorf("cannot determine config path")
	}

	if err := os.MkdirAll(filepath.Dir(path), dirPermissions); err != nil {
		return err
	}

	sections := make(map[string]map[string]string)
	if f, err := os.Open(path); err == nil {
		var parseErr error
		sections, parseErr = parseINI(f)
		_ = f.Close()
		if parseErr != nil {
			return fmt.Errorf("parsing config %s: %w", path, parseErr)
		}
	}

	if _, ok := sections[domain]; !ok {
		sections[domain] = make(map[string]string)
	}
	if token != "" {
		sections[domain]["token"] = token
		delete(sections[domain], "token-cmd")
	}
	if tokenCmd != "" {
		sections[domain]["token-cmd"] = tokenCmd
		delete(sections[domain], "token")
	}
	if forgeType != "" {
		sections[domain]["type"] = forgeType
	}

	return writeINI(path, sections)
}

func writeINI(path string, sections map[string]map[string]string) error {
	var b strings.Builder

	// Write [default] first if it exists.
	if def, ok := sections["default"]; ok {
		writeSection(&b, "default", def)
		delete(sections, "default")
	}

	// Write remaining sections in sorted-ish order (map iteration is fine here).
	for name, kv := range sections {
		writeSection(&b, name, kv)
	}

	if err := os.WriteFile(path, []byte(b.String()), filePermissions); err != nil {
		return err
	}
	// os.WriteFile only applies the mode on creation. Tighten existing files
	// too, since they hold tokens.
	if runtime.GOOS != goosWindows {
		_ = os.Chmod(path, filePermissions)
	}
	return nil
}

func writeSection(b *strings.Builder, name string, kv map[string]string) {
	fmt.Fprintf(b, "[%s]\n", name)
	for k, v := range kv {
		fmt.Fprintf(b, "%s = %s\n", k, v)
	}
	b.WriteString("\n")
}

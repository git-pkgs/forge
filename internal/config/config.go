package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	dirPermissions  = 0700
	filePermissions = 0600
)

type Config struct {
	Default DefaultSection
	Domains map[string]DomainSection
}

type DefaultSection struct {
	Output    string // table, json, plain
	ForgeType string // default forge type
}

type DomainSection struct {
	Type  string // github, gitlab, gitea, forgejo
	Token string // only from user config, never .forge
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
	}

	for name, kv := range sections {
		if name == "default" {
			continue
		}
		ds := cfg.Domains[name]
		if v, ok := kv["type"]; ok {
			ds.Type = v
		}
		if allowTokens {
			if v, ok := kv["token"]; ok {
				ds.Token = v
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
func SetDomain(domain, token, forgeType string) error {
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

	return os.WriteFile(path, []byte(b.String()), filePermissions)
}

func writeSection(b *strings.Builder, name string, kv map[string]string) {
	fmt.Fprintf(b, "[%s]\n", name)
	for k, v := range kv {
		fmt.Fprintf(b, "%s = %s\n", k, v)
	}
	b.WriteString("\n")
}

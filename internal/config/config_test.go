package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseINI(t *testing.T) {
	input := `
# This is a comment
; This is also a comment

[default]
output = json
forge-type = gitlab

[github.com]
token = ghp_abc123

[gitea.example.com]
type = gitea
token = abc123
`

	sections, err := parseINI(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sections["default"]["output"] != "json" {
		t.Errorf("expected output=json, got %q", sections["default"]["output"])
	}
	if sections["default"]["forge-type"] != "gitlab" {
		t.Errorf("expected forge-type=gitlab, got %q", sections["default"]["forge-type"])
	}
	if sections["github.com"]["token"] != "ghp_abc123" {
		t.Errorf("expected github.com token=ghp_abc123, got %q", sections["github.com"]["token"])
	}
	if sections["gitea.example.com"]["type"] != "gitea" {
		t.Errorf("expected gitea.example.com type=gitea, got %q", sections["gitea.example.com"]["type"])
	}
	if sections["gitea.example.com"]["token"] != "abc123" {
		t.Errorf("expected gitea.example.com token=abc123, got %q", sections["gitea.example.com"]["token"])
	}
}

func TestParseINIEmpty(t *testing.T) {
	sections, err := parseINI(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sections) != 0 {
		t.Errorf("expected empty sections, got %d", len(sections))
	}
}

func TestParseINIKeyBeforeSection(t *testing.T) {
	input := `output = table`

	sections, err := parseINI(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sections["default"]["output"] != "table" {
		t.Errorf("expected key before section to land in default, got %v", sections)
	}
}

func TestParseINISpacesAroundEquals(t *testing.T) {
	input := `[test]
key  =  value with spaces
`

	sections, err := parseINI(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sections["test"]["key"] != "value with spaces" {
		t.Errorf("expected trimmed key and value, got key=%q value=%q", "key", sections["test"]["key"])
	}
}

func TestLoadMergesUserAndProject(t *testing.T) {
	ResetCache()
	defer ResetCache()

	dir := t.TempDir()
	userDir := filepath.Join(dir, "usercfg", "forge")
	os.MkdirAll(userDir, 0700)
	userConfig := filepath.Join(userDir, "config")

	os.WriteFile(userConfig, []byte(`
[default]
output = json

[github.com]
token = ghp_user

[gitea.example.com]
type = gitea
token = gitea_tok
`), 0600)

	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0700)
	projectConfig := filepath.Join(projectDir, ".forge")
	os.WriteFile(projectConfig, []byte(`
[default]
forge-type = gitlab

[gitea.example.com]
type = forgejo
token = should_be_ignored
`), 0644)

	// Load manually instead of using Load() since we need custom paths
	cfg := &Config{Domains: make(map[string]DomainSection)}
	if err := loadFile(cfg, userConfig, true); err != nil {
		t.Fatalf("loading user config: %v", err)
	}
	if err := loadFile(cfg, projectConfig, false); err != nil {
		t.Fatalf("loading project config: %v", err)
	}

	// User config sets output
	if cfg.Default.Output != "json" {
		t.Errorf("expected output=json, got %q", cfg.Default.Output)
	}

	// Project config sets forge-type
	if cfg.Default.ForgeType != "gitlab" {
		t.Errorf("expected forge-type=gitlab, got %q", cfg.Default.ForgeType)
	}

	// User config token preserved
	if cfg.Domains["github.com"].Token != "ghp_user" {
		t.Errorf("expected github.com token from user config, got %q", cfg.Domains["github.com"].Token)
	}

	// Project config overrides type but not token
	ds := cfg.Domains["gitea.example.com"]
	if ds.Type != "forgejo" {
		t.Errorf("expected project config to override type to forgejo, got %q", ds.Type)
	}
	if ds.Token != "gitea_tok" {
		t.Errorf("expected token from user config only (not overwritten by project), got %q", ds.Token)
	}
}

func TestProjectConfigTokensIgnored(t *testing.T) {
	cfg := &Config{Domains: make(map[string]DomainSection)}

	r := strings.NewReader(`
[github.com]
token = should_not_load
type = github
`)

	sections, err := parseINI(r)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate loading as project config (allowTokens=false)
	for name, kv := range sections {
		if name == "default" {
			continue
		}
		ds := cfg.Domains[name]
		if v, ok := kv["type"]; ok {
			ds.Type = v
		}
		// Token intentionally not loaded
		cfg.Domains[name] = ds
	}

	if cfg.Domains["github.com"].Token != "" {
		t.Errorf("project config should not load tokens, got %q", cfg.Domains["github.com"].Token)
	}
	if cfg.Domains["github.com"].Type != "github" {
		t.Errorf("expected type=github, got %q", cfg.Domains["github.com"].Type)
	}
}

func TestFindProjectConfig(t *testing.T) {
	dir := t.TempDir()

	// Create nested directories
	nested := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(nested, 0700)

	// No .forge file yet
	got := findProjectConfig(nested)
	if got != "" {
		t.Errorf("expected empty path with no .forge, got %q", got)
	}

	// Create .forge in the middle
	forgePath := filepath.Join(dir, "a", ".forge")
	os.WriteFile(forgePath, []byte("[default]\n"), 0644)

	got = findProjectConfig(nested)
	if got != forgePath {
		t.Errorf("expected %q, got %q", forgePath, got)
	}
}

func TestMissingFilesReturnEmptyConfig(t *testing.T) {
	cfg := &Config{Domains: make(map[string]DomainSection)}
	err := loadFile(cfg, "/nonexistent/path/config", true)
	if err != nil {
		t.Errorf("missing file should not error, got: %v", err)
	}
	if len(cfg.Domains) != 0 {
		t.Errorf("expected no domains, got %d", len(cfg.Domains))
	}
}

func TestUserConfigPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/testxdg")
	got := UserConfigPath()
	want := "/tmp/testxdg/forge/config"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestUserConfigPathDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	got := UserConfigPath()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "forge", "config")
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestSetDomain(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	err := SetDomain("gitea.example.com", "tok123", "gitea")
	if err != nil {
		t.Fatalf("SetDomain: %v", err)
	}

	// Read back
	path := filepath.Join(dir, "forge", "config")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "[gitea.example.com]") {
		t.Error("expected section header in config")
	}
	if !strings.Contains(content, "token = tok123") {
		t.Error("expected token in config")
	}
	if !strings.Contains(content, "type = gitea") {
		t.Error("expected type in config")
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 permissions, got %o", info.Mode().Perm())
	}
}

func TestSetDomainUpdatesExisting(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	// Write initial config
	cfgDir := filepath.Join(dir, "forge")
	os.MkdirAll(cfgDir, 0700)
	os.WriteFile(filepath.Join(cfgDir, "config"), []byte(`[github.com]
token = old_token
`), 0600)

	// Update
	err := SetDomain("github.com", "new_token", "")
	if err != nil {
		t.Fatalf("SetDomain: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(cfgDir, "config"))
	content := string(data)
	if !strings.Contains(content, "token = new_token") {
		t.Errorf("expected updated token, got:\n%s", content)
	}
	if strings.Contains(content, "old_token") {
		t.Error("old token should be replaced")
	}
}

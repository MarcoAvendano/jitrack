// Package config loads and writes jitrack configuration.
//
// Configuration is JSON in two layers — global ~/.config/jitrack/config.json
// and per-repo .jitrack.json — merged so that repo values override global
// ones, and env vars (JITRACK_JIRA_TOKEN, JITRACK_GITHUB_TOKEN) override both.
// Keys are addressed with dotted paths (e.g. "jira.url") both in the
// `jitrack config` commands and internally.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	GlobalFileName = "config.json"
	RepoFileName   = ".jitrack.json"
)

// staticKeys are the fixed configuration keys. branch_prefixes.* and
// transitions.* accept arbitrary sub-keys (Jira issue types / actions).
var staticKeys = []string{
	"provider",
	"jira.url",
	"jira.email",
	"jira.token",
	"github.token",
	"github.api_url",
	"github.owner",
	"github.repo",
	"gitlab.token",
	"gitlab.api_url",
	"gitlab.owner",
	"gitlab.repo",
	"bitbucket.token",
	"bitbucket.username",
	"bitbucket.api_url",
	"bitbucket.owner",
	"bitbucket.repo",
	"base_branch",
}

var dynamicPrefixes = []string{"branch_prefixes.", "transitions."}

var defaults = map[string]string{
	"provider":                "github",
	"base_branch":             "main",
	"github.api_url":          "https://api.github.com",
	"gitlab.api_url":          "https://gitlab.com/api/v4",
	"bitbucket.api_url":       "https://api.bitbucket.org/2.0",
	"branch_prefixes.default": "feature",
	"branch_prefixes.Bug":     "fix",
	"transitions.start":       "In Progress",
	"transitions.close":       "Ready to QA",
}

var envKeys = map[string]string{
	"JITRACK_JIRA_TOKEN":      "jira.token",
	"JITRACK_GITHUB_TOKEN":    "github.token",
	"JITRACK_GITLAB_TOKEN":    "gitlab.token",
	"JITRACK_BITBUCKET_TOKEN": "bitbucket.token",
}

// Config is the merged view of all layers. Sources records, per key,
// where the effective value came from (default | global | repo | env).
type Config struct {
	values  map[string]string
	sources map[string]string
}

// ValidKey reports whether key is settable/gettable.
func ValidKey(key string) bool {
	for _, k := range staticKeys {
		if key == k {
			return true
		}
	}
	for _, p := range dynamicPrefixes {
		if strings.HasPrefix(key, p) && len(key) > len(p) {
			return true
		}
	}
	return false
}

// KnownKeys returns the documented keys, for error messages.
func KnownKeys() []string {
	keys := append([]string{}, staticKeys...)
	keys = append(keys, "branch_prefixes.<IssueType>", "transitions.start", "transitions.close")
	return keys
}

// GlobalPath returns the global config file path, honoring XDG_CONFIG_HOME.
func GlobalPath() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "jitrack", GlobalFileName), nil
}

// Load merges defaults, the global file, the repo file (rooted at repoDir,
// typically the git top-level; pass "" to skip), and env overrides.
func Load(repoDir string) (*Config, error) {
	c := &Config{values: map[string]string{}, sources: map[string]string{}}
	c.overlay(defaults, "default")

	globalPath, err := GlobalPath()
	if err != nil {
		return nil, err
	}
	global, err := readFile(globalPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", globalPath, err)
	}
	c.overlay(global, "global")

	if repoDir != "" {
		repoPath := filepath.Join(repoDir, RepoFileName)
		repo, err := readFile(repoPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", repoPath, err)
		}
		c.overlay(repo, "repo")
	}

	for env, key := range envKeys {
		if v := os.Getenv(env); v != "" {
			c.values[key] = v
			c.sources[key] = "env"
		}
	}
	return c, nil
}

func (c *Config) overlay(vals map[string]string, source string) {
	for k, v := range vals {
		c.values[k] = v
		c.sources[k] = source
	}
}

// Get returns the effective value for a dotted key ("" if unset).
func (c *Config) Get(key string) string { return c.values[key] }

// Source returns which layer the effective value came from.
func (c *Config) Source(key string) string { return c.sources[key] }

// Keys returns all set keys, sorted.
func (c *Config) Keys() []string {
	keys := make([]string, 0, len(c.values))
	for k := range c.values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// BranchPrefix resolves the branch prefix for a Jira issue type.
func (c *Config) BranchPrefix(issueType string) string {
	if p := c.Get("branch_prefixes." + issueType); p != "" {
		return p
	}
	return c.Get("branch_prefixes.default")
}

// RequireJira validates that the Jira connection is configured.
func (c *Config) RequireJira() error {
	var missing []string
	for _, k := range []string{"jira.url", "jira.email", "jira.token"} {
		if c.Get(k) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("jira connection not configured (missing %s) — run `jitrack init` or `jitrack config set <key> <value>`", strings.Join(missing, ", "))
	}
	return nil
}

// Provider returns the configured git provider (defaults to "github").
func (c *Config) Provider() string {
	if p := c.Get("provider"); p != "" {
		return p
	}
	return "github"
}

// RequireProvider validates that the active git provider's connection is
// configured (i.e. its token is set).
func (c *Config) RequireProvider() error {
	p := c.Provider()
	if c.Get(p+".token") == "" {
		return fmt.Errorf("%s connection not configured (missing %s.token) — run `jitrack init` or `jitrack config set %s.token <token>`", p, p, p)
	}
	return nil
}

// Set writes key=value into the JSON file at path, creating the file and
// parent directory as needed. The file keeps its nested JSON shape.
func Set(path, key, value string) error {
	if !ValidKey(key) {
		return fmt.Errorf("unknown config key %q — valid keys: %s", key, strings.Join(KnownKeys(), ", "))
	}
	raw, err := readRaw(path)
	if err != nil {
		return err
	}
	setNested(raw, strings.Split(key, "."), value)
	return writeRaw(path, raw)
}

// readFile reads a config JSON file into flat dotted keys.
// A missing file is not an error — it returns an empty map.
func readFile(path string) (map[string]string, error) {
	raw, err := readRaw(path)
	if err != nil {
		return nil, err
	}
	flat := map[string]string{}
	flatten("", raw, flat)
	return flat, nil
}

func readRaw(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", path, err)
	}
	return raw, nil
}

func writeRaw(path string, raw map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	// 0600: the global file holds API tokens.
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func flatten(prefix string, node map[string]any, out map[string]string) {
	for k, v := range node {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]any:
			flatten(key, val, out)
		case string:
			out[key] = val
		default:
			out[key] = fmt.Sprint(val)
		}
	}
}

func setNested(node map[string]any, path []string, value string) {
	if len(path) == 1 {
		node[path[0]] = value
		return
	}
	child, ok := node[path[0]].(map[string]any)
	if !ok {
		child = map[string]any{}
		node[path[0]] = child
	}
	setNested(child, path[1:], value)
}

// Mask hides secret values for display.
func Mask(key, value string) string {
	if value == "" || !strings.Contains(key, "token") {
		return value
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "…" + value[len(value)-4:]
}

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// point the global config at a temp dir for the duration of a test.
func tempGlobal(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return filepath.Join(dir, "sr-cli", GlobalFileName)
}

func TestSetAndLoadMerge(t *testing.T) {
	globalPath := tempGlobal(t)
	repoDir := t.TempDir()

	if err := Set(globalPath, "jira.url", "https://example.atlassian.net"); err != nil {
		t.Fatal(err)
	}
	if err := Set(globalPath, "base_branch", "main"); err != nil {
		t.Fatal(err)
	}
	// repo layer overrides global
	repoPath := filepath.Join(repoDir, RepoFileName)
	if err := Set(repoPath, "base_branch", "develop"); err != nil {
		t.Fatal(err)
	}

	c, err := Load(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Get("jira.url"); got != "https://example.atlassian.net" {
		t.Errorf("jira.url = %q", got)
	}
	if got := c.Get("base_branch"); got != "develop" {
		t.Errorf("base_branch = %q, want repo override 'develop'", got)
	}
	if got := c.Source("base_branch"); got != "repo" {
		t.Errorf("source(base_branch) = %q, want 'repo'", got)
	}
	if got := c.Get("github.api_url"); got != "https://api.github.com" {
		t.Errorf("default github.api_url = %q", got)
	}
}

func TestEnvOverride(t *testing.T) {
	tempGlobal(t)
	t.Setenv("SR_JIRA_TOKEN", "env-token")
	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Get("jira.token"); got != "env-token" {
		t.Errorf("jira.token = %q, want env override", got)
	}
	if got := c.Source("jira.token"); got != "env" {
		t.Errorf("source = %q, want 'env'", got)
	}
}

func TestSetRejectsUnknownKey(t *testing.T) {
	globalPath := tempGlobal(t)
	if err := Set(globalPath, "nope.bad", "x"); err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestValidKey(t *testing.T) {
	for _, k := range []string{"jira.url", "github.token", "base_branch", "branch_prefixes.Bug", "transitions.start"} {
		if !ValidKey(k) {
			t.Errorf("ValidKey(%q) = false, want true", k)
		}
	}
	for _, k := range []string{"jira", "branch_prefixes.", "random", "jira.password"} {
		if ValidKey(k) {
			t.Errorf("ValidKey(%q) = true, want false", k)
		}
	}
}

func TestGlobalFilePermissions(t *testing.T) {
	globalPath := tempGlobal(t)
	if err := Set(globalPath, "jira.token", "secret"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(globalPath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file perms = %o, want 600", perm)
	}
}

func TestBranchPrefix(t *testing.T) {
	tempGlobal(t)
	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if got := c.BranchPrefix("Bug"); got != "fix" {
		t.Errorf("BranchPrefix(Bug) = %q", got)
	}
	if got := c.BranchPrefix("Story"); got != "feature" {
		t.Errorf("BranchPrefix(Story) = %q, want default 'feature'", got)
	}
}

func TestMask(t *testing.T) {
	if got := Mask("jira.token", "abcd1234efgh5678"); got != "abcd…5678" {
		t.Errorf("Mask = %q", got)
	}
	if got := Mask("jira.token", "short"); got != "****" {
		t.Errorf("Mask short = %q", got)
	}
	if got := Mask("jira.url", "https://x"); got != "https://x" {
		t.Errorf("Mask non-token = %q", got)
	}
}

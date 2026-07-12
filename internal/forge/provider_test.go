package forge

import (
	"path/filepath"
	"testing"

	"github.com/MarcoAvendano/jitrack/internal/config"
)

// loadWith writes keys to a temp global config and loads it.
func loadWith(t *testing.T, kv map[string]string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "jitrack", config.GlobalFileName)
	for k, v := range kv {
		if err := config.Set(path, k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestNewSelectsProvider(t *testing.T) {
	remote := "git@example.com:owner/repo.git"

	cfg := loadWith(t, map[string]string{"github.token": "t"})
	p, err := New(cfg, remote)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "github" {
		t.Errorf("default provider = %q, want github", p.Name())
	}

	cfg = loadWith(t, map[string]string{"provider": "gitlab", "gitlab.token": "t"})
	p, err = New(cfg, remote)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "gitlab" {
		t.Errorf("provider = %q, want gitlab", p.Name())
	}

	cfg = loadWith(t, map[string]string{"provider": "bitbucket", "bitbucket.username": "u", "bitbucket.token": "t"})
	p, err = New(cfg, remote)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "bitbucket" {
		t.Errorf("provider = %q, want bitbucket", p.Name())
	}

	cfg = loadWith(t, map[string]string{"provider": "bogus"})
	if _, err := New(cfg, remote); err == nil {
		t.Error("New with unknown provider should error")
	}
}

func TestResolveRepoConfigOverElseRemote(t *testing.T) {
	parse := func(url string) (string, string, error) { return "fromremote", "repo", nil }

	cfg := loadWith(t, map[string]string{"github.owner": "cfgowner", "github.repo": "cfgrepo"})
	owner, repo := resolveRepo(cfg, "github", "git@x:y/z.git", parse)
	if owner != "cfgowner" || repo != "cfgrepo" {
		t.Errorf("config override = %q/%q", owner, repo)
	}

	cfg = loadWith(t, nil)
	owner, repo = resolveRepo(cfg, "github", "git@x:y/z.git", parse)
	if owner != "fromremote" || repo != "repo" {
		t.Errorf("remote fallback = %q/%q", owner, repo)
	}

	// No config, no remote -> empty (Viewer can still run).
	owner, repo = resolveRepo(cfg, "github", "", parse)
	if owner != "" || repo != "" {
		t.Errorf("empty remote should yield empty owner/repo, got %q/%q", owner, repo)
	}
}

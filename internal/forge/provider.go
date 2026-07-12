// Package forge abstracts the git hosting provider (GitHub, GitLab, …) behind
// a single Provider interface so the commands don't care which one is in use.
//
// A provider is built by the New factory, which reads the "provider" config key
// (default "github") and returns the matching implementation with owner/repo
// baked in. GitLab's merge requests and GitHub's pull requests are both mapped
// onto the neutral PullRequest value type.
package forge

import (
	"fmt"
	"strings"

	"github.com/MarcoAvendano/jitrack/internal/bitbucket"
	"github.com/MarcoAvendano/jitrack/internal/config"
	"github.com/MarcoAvendano/jitrack/internal/github"
	"github.com/MarcoAvendano/jitrack/internal/gitlab"
)

// PullRequest is the provider-neutral view of a pull/merge request.
type PullRequest struct {
	Number  int
	URL     string
	Title   string
	State   string // "open" | "closed"
	Merged  bool
	HeadRef string // the source/head branch
}

// Provider is the set of git-host operations jitrack needs. Owner/repo are
// baked into the instance, so callers work in terms of branches only.
type Provider interface {
	// Name reports the provider identifier ("github" | "gitlab").
	Name() string
	// Viewer validates credentials and returns the authenticated login.
	// It does not require owner/repo, so it works during `jitrack init`.
	Viewer() (string, error)
	// FindOpenPR returns the open PR/MR whose head is branch, or nil if none.
	FindOpenPR(branch string) (*PullRequest, error)
	// ListPRs returns PRs/MRs newest first. state is "open"|"closed"|"all".
	ListPRs(state string) ([]PullRequest, error)
	// CreatePR opens a PR/MR from head into base.
	CreatePR(title, head, base, body string) (*PullRequest, error)
}

// New builds the configured provider. remoteURL is the origin remote used to
// derive owner/repo when config doesn't set them explicitly; pass "" when it's
// unavailable (e.g. during `jitrack init` — Viewer needs no repo).
func New(cfg *config.Config, remoteURL string) (Provider, error) {
	switch name := cfg.Provider(); name {
	case "github":
		return newGitHub(cfg, remoteURL), nil
	case "gitlab":
		return newGitLab(cfg, remoteURL), nil
	case "bitbucket":
		return newBitbucket(cfg, remoteURL), nil
	default:
		return nil, fmt.Errorf("unknown provider %q — valid providers: %s", name, strings.Join(Providers(), ", "))
	}
}

// Providers lists the supported provider identifiers.
func Providers() []string { return []string{"github", "gitlab", "bitbucket"} }

// Validate checks credentials for a provider without needing a repo, returning
// the authenticated login. Used by `jitrack init` before persisting config.
// username is only consulted for providers that use Basic auth (bitbucket);
// github and gitlab ignore it.
func Validate(provider, apiURL, username, token string) (string, error) {
	switch provider {
	case "github":
		return github.NewClient(apiURL, token).Viewer()
	case "gitlab":
		return gitlab.NewClient(apiURL, token).Viewer()
	case "bitbucket":
		return bitbucket.NewClient(apiURL, username, token).Viewer()
	default:
		return "", fmt.Errorf("unknown provider %q — valid providers: %s", provider, strings.Join(Providers(), ", "))
	}
}

// resolveRepo returns the owner/repo for a provider: config
// <provider>.owner/<provider>.repo first, else parsed from remoteURL by parse.
// Resolution is best-effort — on failure it returns empty strings so credential
// validation (Viewer) can still run; PR operations guard against empty values.
func resolveRepo(cfg *config.Config, provider, remoteURL string, parse func(string) (string, string, error)) (owner, repo string) {
	owner, repo = cfg.Get(provider+".owner"), cfg.Get(provider+".repo")
	if owner != "" && repo != "" {
		return owner, repo
	}
	if remoteURL == "" {
		return "", ""
	}
	o, r, err := parse(remoteURL)
	if err != nil {
		return "", ""
	}
	return o, r
}

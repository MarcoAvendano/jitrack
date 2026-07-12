package forge

import (
	"fmt"

	"github.com/MarcoAvendano/jitrack/internal/config"
	"github.com/MarcoAvendano/jitrack/internal/gitlab"
)

// gitlabProvider adapts the internal/gitlab client to the Provider interface,
// presenting merge requests as neutral PullRequests.
type gitlabProvider struct {
	client      *gitlab.Client
	owner, repo string
}

func newGitLab(cfg *config.Config, remoteURL string) *gitlabProvider {
	owner, repo := resolveRepo(cfg, "gitlab", remoteURL, gitlab.ParseRemoteURL)
	return &gitlabProvider{
		client: gitlab.NewClient(cfg.Get("gitlab.api_url"), cfg.Get("gitlab.token")),
		owner:  owner,
		repo:   repo,
	}
}

func (p *gitlabProvider) Name() string { return "gitlab" }

func (p *gitlabProvider) Viewer() (string, error) { return p.client.Viewer() }

func (p *gitlabProvider) requireRepo() error {
	if p.owner == "" || p.repo == "" {
		return fmt.Errorf("could not determine GitLab project — set gitlab.owner and gitlab.repo or run inside a repo with an origin remote")
	}
	return nil
}

func (p *gitlabProvider) FindOpenPR(branch string) (*PullRequest, error) {
	if err := p.requireRepo(); err != nil {
		return nil, err
	}
	mr, err := p.client.FindOpenMR(p.owner, p.repo, branch)
	if err != nil || mr == nil {
		return nil, err
	}
	return fromGitLabMR(mr), nil
}

func (p *gitlabProvider) ListPRs(state string) ([]PullRequest, error) {
	if err := p.requireRepo(); err != nil {
		return nil, err
	}
	mrs, err := p.client.ListMRs(p.owner, p.repo, gitlabState(state))
	if err != nil {
		return nil, err
	}
	out := make([]PullRequest, len(mrs))
	for i := range mrs {
		out[i] = *fromGitLabMR(&mrs[i])
	}
	return out, nil
}

func (p *gitlabProvider) CreatePR(title, head, base, body string) (*PullRequest, error) {
	if err := p.requireRepo(); err != nil {
		return nil, err
	}
	mr, err := p.client.CreateMR(p.owner, p.repo, title, head, base, body)
	if err != nil {
		return nil, err
	}
	return fromGitLabMR(mr), nil
}

// gitlabState maps the neutral state ("open"|"closed"|"all") to GitLab's.
func gitlabState(state string) string {
	if state == "open" {
		return "opened"
	}
	return state // "closed" and "all" are the same word in GitLab
}

func fromGitLabMR(mr *gitlab.MergeRequest) *PullRequest {
	state := "closed"
	if mr.State == "opened" {
		state = "open"
	}
	return &PullRequest{
		Number:  mr.IID,
		URL:     mr.WebURL,
		Title:   mr.Title,
		State:   state,
		Merged:  mr.Merged(),
		HeadRef: mr.SourceBranch,
	}
}

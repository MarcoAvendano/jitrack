package forge

import (
	"fmt"

	"github.com/MarcoAvendano/jitrack/internal/config"
	"github.com/MarcoAvendano/jitrack/internal/github"
)

// githubProvider adapts the internal/github client to the Provider interface.
type githubProvider struct {
	client      *github.Client
	owner, repo string
}

func newGitHub(cfg *config.Config, remoteURL string) *githubProvider {
	owner, repo := resolveRepo(cfg, "github", remoteURL, github.ParseRemoteURL)
	return &githubProvider{
		client: github.NewClient(cfg.Get("github.api_url"), cfg.Get("github.token")),
		owner:  owner,
		repo:   repo,
	}
}

func (p *githubProvider) Name() string { return "github" }

func (p *githubProvider) Viewer() (string, error) { return p.client.Viewer() }

func (p *githubProvider) requireRepo() error {
	if p.owner == "" || p.repo == "" {
		return fmt.Errorf("could not determine GitHub owner/repo — set github.owner and github.repo or run inside a repo with an origin remote")
	}
	return nil
}

func (p *githubProvider) FindOpenPR(branch string) (*PullRequest, error) {
	if err := p.requireRepo(); err != nil {
		return nil, err
	}
	pr, err := p.client.FindOpenPR(p.owner, p.repo, branch)
	if err != nil || pr == nil {
		return nil, err
	}
	return fromGitHubPR(pr), nil
}

func (p *githubProvider) ListPRs(state string) ([]PullRequest, error) {
	if err := p.requireRepo(); err != nil {
		return nil, err
	}
	prs, err := p.client.ListPRs(p.owner, p.repo, state)
	if err != nil {
		return nil, err
	}
	out := make([]PullRequest, len(prs))
	for i := range prs {
		out[i] = *fromGitHubPR(&prs[i])
	}
	return out, nil
}

func (p *githubProvider) CreatePR(title, head, base, body string) (*PullRequest, error) {
	if err := p.requireRepo(); err != nil {
		return nil, err
	}
	pr, err := p.client.CreatePR(p.owner, p.repo, title, head, base, body)
	if err != nil {
		return nil, err
	}
	return fromGitHubPR(pr), nil
}

func fromGitHubPR(pr *github.PullRequest) *PullRequest {
	return &PullRequest{
		Number:  pr.Number,
		URL:     pr.HTMLURL,
		Title:   pr.Title,
		State:   pr.State,
		Merged:  pr.Merged(),
		HeadRef: pr.Head.Ref,
	}
}

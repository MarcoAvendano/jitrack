package forge

import (
	"fmt"

	"github.com/MarcoAvendano/jitrack/internal/bitbucket"
	"github.com/MarcoAvendano/jitrack/internal/config"
)

// bitbucketProvider adapts the internal/bitbucket client to the Provider
// interface. owner is the Bitbucket workspace; repo is the repo slug.
type bitbucketProvider struct {
	client      *bitbucket.Client
	owner, repo string
}

func newBitbucket(cfg *config.Config, remoteURL string) *bitbucketProvider {
	owner, repo := resolveRepo(cfg, "bitbucket", remoteURL, bitbucket.ParseRemoteURL)
	return &bitbucketProvider{
		client: bitbucket.NewClient(cfg.Get("bitbucket.api_url"), cfg.Get("bitbucket.username"), cfg.Get("bitbucket.token")),
		owner:  owner,
		repo:   repo,
	}
}

func (p *bitbucketProvider) Name() string { return "bitbucket" }

func (p *bitbucketProvider) Viewer() (string, error) { return p.client.Viewer() }

func (p *bitbucketProvider) requireRepo() error {
	if p.owner == "" || p.repo == "" {
		return fmt.Errorf("could not determine Bitbucket workspace/repo — set bitbucket.owner and bitbucket.repo or run inside a repo with an origin remote")
	}
	return nil
}

func (p *bitbucketProvider) FindOpenPR(branch string) (*PullRequest, error) {
	if err := p.requireRepo(); err != nil {
		return nil, err
	}
	pr, err := p.client.FindOpenPR(p.owner, p.repo, branch)
	if err != nil || pr == nil {
		return nil, err
	}
	return fromBitbucketPR(pr), nil
}

func (p *bitbucketProvider) ListPRs(state string) ([]PullRequest, error) {
	if err := p.requireRepo(); err != nil {
		return nil, err
	}
	prs, err := p.client.ListPRs(p.owner, p.repo, bitbucketStates(state)...)
	if err != nil {
		return nil, err
	}
	out := make([]PullRequest, len(prs))
	for i := range prs {
		out[i] = *fromBitbucketPR(&prs[i])
	}
	return out, nil
}

func (p *bitbucketProvider) CreatePR(title, head, base, body string) (*PullRequest, error) {
	if err := p.requireRepo(); err != nil {
		return nil, err
	}
	pr, err := p.client.CreatePR(p.owner, p.repo, title, head, base, body)
	if err != nil {
		return nil, err
	}
	return fromBitbucketPR(pr), nil
}

// bitbucketStates maps the neutral state ("open"|"closed"|"all") to the
// Bitbucket PR states to query. Bitbucket has no single "closed" state, so it
// expands to the three non-open ones.
func bitbucketStates(state string) []string {
	switch state {
	case "open":
		return []string{"OPEN"}
	case "closed":
		return []string{"MERGED", "DECLINED", "SUPERSEDED"}
	default: // "all"
		return []string{"OPEN", "MERGED", "DECLINED", "SUPERSEDED"}
	}
}

func fromBitbucketPR(pr *bitbucket.PullRequest) *PullRequest {
	state := "closed"
	if pr.State == "OPEN" {
		state = "open"
	}
	return &PullRequest{
		Number:  pr.ID,
		URL:     pr.URL(),
		Title:   pr.Title,
		State:   state,
		Merged:  pr.Merged(),
		HeadRef: pr.HeadRef(),
	}
}

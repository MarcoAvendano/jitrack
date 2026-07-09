// Package github is a thin client for the GitHub REST API, covering only
// what sr-cli needs: find and create pull requests.
package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Client struct {
	APIURL string // e.g. https://api.github.com
	Token  string
	http   *http.Client
}

func NewClient(apiURL, token string) *Client {
	return &Client{
		APIURL: strings.TrimRight(apiURL, "/"),
		Token:  token,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

type PullRequest struct {
	Number   int    `json:"number"`
	HTMLURL  string `json:"html_url"`
	Title    string `json:"title"`
	State    string `json:"state"`     // open | closed
	MergedAt string `json:"merged_at"` // empty when not merged
	Head     struct {
		Ref string `json:"ref"`
	} `json:"head"`
}

// Merged reports whether a closed PR was merged (vs. just closed).
func (pr *PullRequest) Merged() bool { return pr.MergedAt != "" }

var remoteRe = regexp.MustCompile(`^(?:git@[^:]+:|https://[^/]+/|ssh://git@[^/]+/)([^/]+)/(.+?)(?:\.git)?$`)

// ParseRemoteURL extracts owner and repo from an SSH or HTTPS git remote,
// e.g. git@github.com:my-org/my-repo.git or
// https://github.com/my-org/my-repo.
func ParseRemoteURL(url string) (owner, repo string, err error) {
	m := remoteRe.FindStringSubmatch(strings.TrimSpace(url))
	if m == nil {
		return "", "", fmt.Errorf("cannot parse owner/repo from remote URL %q", url)
	}
	return m[1], m[2], nil
}

func (c *Client) do(method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.APIURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("github request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("github authentication failed — check github.token")
		}
		return fmt.Errorf("github %s %s: HTTP %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// Viewer validates credentials; returns the authenticated login.
func (c *Client) Viewer() (string, error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := c.do("GET", "/user", nil, &user); err != nil {
		return "", err
	}
	return user.Login, nil
}

// FindOpenPR returns the open PR whose head is branch, or nil if none.
func (c *Client) FindOpenPR(owner, repo, branch string) (*PullRequest, error) {
	var prs []PullRequest
	path := fmt.Sprintf("/repos/%s/%s/pulls?state=open&head=%s:%s", owner, repo, owner, branch)
	if err := c.do("GET", path, nil, &prs); err != nil {
		return nil, err
	}
	if len(prs) == 0 {
		return nil, nil
	}
	return &prs[0], nil
}

// ListPRs returns the repo's pull requests, newest first.
// state is "open", "closed", or "all".
func (c *Client) ListPRs(owner, repo, state string) ([]PullRequest, error) {
	var prs []PullRequest
	path := fmt.Sprintf("/repos/%s/%s/pulls?state=%s&per_page=100&sort=created&direction=desc", owner, repo, state)
	if err := c.do("GET", path, nil, &prs); err != nil {
		return nil, err
	}
	return prs, nil
}

// CreatePR opens a pull request from head into base.
func (c *Client) CreatePR(owner, repo, title, head, base, body string) (*PullRequest, error) {
	payload := map[string]string{
		"title": title,
		"head":  head,
		"base":  base,
		"body":  body,
	}
	var pr PullRequest
	path := fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)
	if err := c.do("POST", path, payload, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

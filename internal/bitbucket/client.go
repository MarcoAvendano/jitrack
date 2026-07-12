// Package bitbucket is a thin client for the Bitbucket Cloud REST v2 API,
// covering only what jitrack needs: find and create pull requests. It mirrors
// the shape of internal/github so the two slot behind the same provider
// abstraction.
//
// Bitbucket Cloud authenticates with HTTP Basic auth: the username (your
// Bitbucket username or Atlassian email) plus an app password / API token as
// the password. Workspace/repo access tokens (Bearer) are intentionally not
// used because they cannot resolve /user, which `jitrack init` validates.
package bitbucket

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Client struct {
	APIURL   string // e.g. https://api.bitbucket.org/2.0
	Username string
	Token    string // app password or API token
	http     *http.Client
}

func NewClient(apiURL, username, token string) *Client {
	return &Client{
		APIURL:   strings.TrimRight(apiURL, "/"),
		Username: username,
		Token:    token,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

// PullRequest is the subset of a Bitbucket PR jitrack reads.
type PullRequest struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	State string `json:"state"` // OPEN | MERGED | DECLINED | SUPERSEDED
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"source"`
}

// Merged reports whether the PR was merged (vs. declined/superseded/open).
func (pr *PullRequest) Merged() bool { return pr.State == "MERGED" }

// URL returns the browser URL for the PR.
func (pr *PullRequest) URL() string { return pr.Links.HTML.Href }

// HeadRef returns the PR's source branch.
func (pr *PullRequest) HeadRef() string { return pr.Source.Branch.Name }

// page is the paginated envelope Bitbucket wraps list responses in.
type page struct {
	Values []PullRequest `json:"values"`
}

var remoteRe = regexp.MustCompile(`^(?:git@[^:]+:|https://[^/]+/|ssh://git@[^/]+/)([^/]+)/(.+?)(?:\.git)?$`)

// ParseRemoteURL extracts workspace and repo slug from an SSH or HTTPS git
// remote, e.g. git@bitbucket.org:my-workspace/my-repo.git.
func ParseRemoteURL(u string) (workspace, repo string, err error) {
	m := remoteRe.FindStringSubmatch(strings.TrimSpace(u))
	if m == nil {
		return "", "", fmt.Errorf("cannot parse workspace/repo from remote URL %q", u)
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
	basic := base64.StdEncoding.EncodeToString([]byte(c.Username + ":" + c.Token))
	req.Header.Set("Authorization", "Basic "+basic)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("bitbucket request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("bitbucket authentication failed — check bitbucket.username and bitbucket.token")
		}
		return fmt.Errorf("bitbucket %s %s: HTTP %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// Viewer validates credentials; returns the authenticated account's nickname
// (falling back to its display name).
func (c *Client) Viewer() (string, error) {
	var user struct {
		Nickname    string `json:"nickname"`
		DisplayName string `json:"display_name"`
	}
	if err := c.do("GET", "/user", nil, &user); err != nil {
		return "", err
	}
	if user.Nickname != "" {
		return user.Nickname, nil
	}
	return user.DisplayName, nil
}

// FindOpenPR returns the open PR whose source branch is branch, or nil if none.
func (c *Client) FindOpenPR(workspace, repo, branch string) (*PullRequest, error) {
	q := fmt.Sprintf(`source.branch.name="%s"`, branch)
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests?state=OPEN&q=%s", workspace, repo, url.QueryEscape(q))
	var p page
	if err := c.do("GET", path, nil, &p); err != nil {
		return nil, err
	}
	if len(p.Values) == 0 {
		return nil, nil
	}
	return &p.Values[0], nil
}

// ListPRs returns the repo's pull requests in the given states, newest first.
// With no states it defaults to OPEN. Valid states: OPEN, MERGED, DECLINED,
// SUPERSEDED.
func (c *Client) ListPRs(workspace, repo string, states ...string) ([]PullRequest, error) {
	q := url.Values{}
	for _, s := range states {
		q.Add("state", s)
	}
	q.Set("pagelen", "50")
	q.Set("sort", "-created_on")
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests?%s", workspace, repo, q.Encode())
	var p page
	if err := c.do("GET", path, nil, &p); err != nil {
		return nil, err
	}
	return p.Values, nil
}

// CreatePR opens a pull request from source into destination.
func (c *Client) CreatePR(workspace, repo, title, source, destination, description string) (*PullRequest, error) {
	type branch struct {
		Name string `json:"name"`
	}
	type ref struct {
		Branch branch `json:"branch"`
	}
	payload := struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Source      ref    `json:"source"`
		Destination ref    `json:"destination"`
	}{
		Title:       title,
		Description: description,
		Source:      ref{Branch: branch{Name: source}},
		Destination: ref{Branch: branch{Name: destination}},
	}
	var pr PullRequest
	path := fmt.Sprintf("/repositories/%s/%s/pullrequests", workspace, repo)
	if err := c.do("POST", path, payload, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

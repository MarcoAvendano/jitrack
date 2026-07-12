// Package gitlab is a thin client for the GitLab REST v4 API, covering only
// what jitrack needs: find and create merge requests. It mirrors the shape of
// internal/github so the two slot behind the same provider abstraction.
package gitlab

import (
	"bytes"
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
	APIURL string // e.g. https://gitlab.com/api/v4
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

// MergeRequest is the subset of a GitLab MR jitrack reads. Note iid is the
// project-scoped number shown in the UI (not the global id).
type MergeRequest struct {
	IID          int    `json:"iid"`
	WebURL       string `json:"web_url"`
	Title        string `json:"title"`
	State        string `json:"state"` // opened | closed | merged | locked
	MergedAt     string `json:"merged_at"`
	SourceBranch string `json:"source_branch"`
}

// Merged reports whether the MR was merged (vs. just closed).
func (mr *MergeRequest) Merged() bool { return mr.State == "merged" || mr.MergedAt != "" }

var remoteRe = regexp.MustCompile(`^(?:git@[^:]+:|https://[^/]+/|ssh://git@[^/]+/)([^/]+)/(.+?)(?:\.git)?$`)

// ParseRemoteURL extracts owner and repo from an SSH or HTTPS git remote. For
// GitLab the "repo" may itself contain slashes (nested subgroups), which the
// project path (owner + "/" + repo) preserves.
func ParseRemoteURL(u string) (owner, repo string, err error) {
	m := remoteRe.FindStringSubmatch(strings.TrimSpace(u))
	if m == nil {
		return "", "", fmt.Errorf("cannot parse owner/repo from remote URL %q", u)
	}
	return m[1], m[2], nil
}

// projectPath builds the URL-encoded "namespace/project" identifier GitLab
// expects in /projects/:id (e.g. group/sub/repo -> group%2Fsub%2Frepo).
func projectPath(owner, repo string) string {
	return url.PathEscape(owner + "/" + repo)
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
	req.Header.Set("PRIVATE-TOKEN", c.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("gitlab request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("gitlab authentication failed — check gitlab.token")
		}
		return fmt.Errorf("gitlab %s %s: HTTP %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// Viewer validates credentials; returns the authenticated username.
func (c *Client) Viewer() (string, error) {
	var user struct {
		Username string `json:"username"`
	}
	if err := c.do("GET", "/user", nil, &user); err != nil {
		return "", err
	}
	return user.Username, nil
}

// FindOpenMR returns the open MR whose source branch is branch, or nil if none.
func (c *Client) FindOpenMR(owner, repo, branch string) (*MergeRequest, error) {
	var mrs []MergeRequest
	path := fmt.Sprintf("/projects/%s/merge_requests?state=opened&source_branch=%s", projectPath(owner, repo), url.QueryEscape(branch))
	if err := c.do("GET", path, nil, &mrs); err != nil {
		return nil, err
	}
	if len(mrs) == 0 {
		return nil, nil
	}
	return &mrs[0], nil
}

// ListMRs returns the project's merge requests, newest first.
// state is "opened", "closed", "merged", or "all".
func (c *Client) ListMRs(owner, repo, state string) ([]MergeRequest, error) {
	var mrs []MergeRequest
	path := fmt.Sprintf("/projects/%s/merge_requests?state=%s&per_page=100&order_by=created_at&sort=desc", projectPath(owner, repo), url.QueryEscape(state))
	if err := c.do("GET", path, nil, &mrs); err != nil {
		return nil, err
	}
	return mrs, nil
}

// CreateMR opens a merge request from source into target.
func (c *Client) CreateMR(owner, repo, title, source, target, description string) (*MergeRequest, error) {
	payload := map[string]string{
		"title":         title,
		"source_branch": source,
		"target_branch": target,
		"description":   description,
	}
	var mr MergeRequest
	path := fmt.Sprintf("/projects/%s/merge_requests", projectPath(owner, repo))
	if err := c.do("POST", path, payload, &mr); err != nil {
		return nil, err
	}
	return &mr, nil
}

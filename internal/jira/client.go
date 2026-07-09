// Package jira is a thin client for the Jira Cloud REST API v3, covering
// only what jitrack needs: read an issue, transition it, comment on it.
package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL string // e.g. https://yourteam.atlassian.net
	Email   string
	Token   string
	http    *http.Client
}

func NewClient(baseURL, email, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Email:   email,
		Token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

type Issue struct {
	Key       string
	Summary   string
	IssueType string
	Status    string
}

type Transition struct {
	ID   string
	Name string
	To   string // status the transition leads to
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
	req, err := http.NewRequest(method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Email, c.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("jira request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return fmt.Errorf("jira authentication failed (%d) — check jira.email and jira.token", resp.StatusCode)
		case http.StatusNotFound:
			return fmt.Errorf("jira: not found (%s)", path)
		}
		return fmt.Errorf("jira %s %s: HTTP %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// Myself validates credentials; returns the account's display name.
func (c *Client) Myself() (string, error) {
	var me struct {
		DisplayName string `json:"displayName"`
	}
	if err := c.do("GET", "/rest/api/3/myself", nil, &me); err != nil {
		return "", err
	}
	return me.DisplayName, nil
}

// GetIssue fetches key's summary, issue type, and status.
func (c *Client) GetIssue(key string) (*Issue, error) {
	var raw struct {
		Key    string `json:"key"`
		Fields struct {
			Summary   string `json:"summary"`
			IssueType struct {
				Name string `json:"name"`
			} `json:"issuetype"`
			Status struct {
				Name string `json:"name"`
			} `json:"status"`
		} `json:"fields"`
	}
	path := fmt.Sprintf("/rest/api/3/issue/%s?fields=summary,issuetype,status", key)
	if err := c.do("GET", path, nil, &raw); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, fmt.Errorf("ticket %s not found in Jira", key)
		}
		return nil, err
	}
	return &Issue{
		Key:       raw.Key,
		Summary:   raw.Fields.Summary,
		IssueType: raw.Fields.IssueType.Name,
		Status:    raw.Fields.Status.Name,
	}, nil
}

// Transitions lists the transitions currently available for the issue.
func (c *Client) Transitions(key string) ([]Transition, error) {
	var raw struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			To   struct {
				Name string `json:"name"`
			} `json:"to"`
		} `json:"transitions"`
	}
	if err := c.do("GET", "/rest/api/3/issue/"+key+"/transitions", nil, &raw); err != nil {
		return nil, err
	}
	ts := make([]Transition, len(raw.Transitions))
	for i, t := range raw.Transitions {
		ts[i] = Transition{ID: t.ID, Name: t.Name, To: t.To.Name}
	}
	return ts, nil
}

// PickTransition finds the transition matching target, first by transition
// name, then by the status it leads to (both case-insensitive). This lets
// config say "In Progress" (a status) even when the board names the
// transition "Start work". Returns nil if nothing matches.
func PickTransition(ts []Transition, target string) *Transition {
	for i := range ts {
		if strings.EqualFold(ts[i].Name, target) {
			return &ts[i]
		}
	}
	for i := range ts {
		if strings.EqualFold(ts[i].To, target) {
			return &ts[i]
		}
	}
	return nil
}

// TransitionTo moves the issue through the transition whose name or target
// status matches (case-insensitive). Lists what's available if none does.
func (c *Client) TransitionTo(key, target string) error {
	ts, err := c.Transitions(key)
	if err != nil {
		return err
	}
	t := PickTransition(ts, target)
	if t == nil {
		var available []string
		for _, t := range ts {
			available = append(available, fmt.Sprintf("%s → %s", t.Name, t.To))
		}
		return fmt.Errorf("no transition named or leading to %q for %s (available: %s)", target, key, strings.Join(available, ", "))
	}
	body := map[string]any{"transition": map[string]string{"id": t.ID}}
	return c.do("POST", "/rest/api/3/issue/"+key+"/transitions", body, nil)
}

// AddComment posts a plain-text comment (wrapped in the ADF document
// format that API v3 requires).
func (c *Client) AddComment(key, text string) error {
	body := map[string]any{
		"body": map[string]any{
			"type":    "doc",
			"version": 1,
			"content": []any{
				map[string]any{
					"type": "paragraph",
					"content": []any{
						map[string]any{"type": "text", "text": text},
					},
				},
			},
		},
	}
	return c.do("POST", "/rest/api/3/issue/"+key+"/comment", body, nil)
}

// BrowseURL returns the human-facing URL for a ticket.
func (c *Client) BrowseURL(key string) string {
	return c.BaseURL + "/browse/" + key
}

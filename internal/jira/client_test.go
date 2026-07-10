package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAssignToMe(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/rest/api/3/myself":
			json.NewEncoder(w).Encode(map[string]string{"accountId": "acc-123"})
		case r.Method == "PUT" && r.URL.Path == "/rest/api/3/issue/KAN-1/assignee":
			json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "me@example.com", "token")
	if err := c.AssignToMe("KAN-1"); err != nil {
		t.Fatal(err)
	}
	if gotBody["accountId"] != "acc-123" {
		t.Errorf("assignee body = %v, want accountId acc-123", gotBody)
	}
}

func TestPickTransition(t *testing.T) {
	// Modeled on the user's board: transition names differ from statuses.
	ts := []Transition{
		{ID: "1", Name: "Ready to work", To: "To Do"},
		{ID: "2", Name: "Start work", To: "In Progress"},
		{ID: "3", Name: "Back to backlog", To: "Backlog"},
	}
	cases := []struct {
		target string
		wantID string
	}{
		{"Start work", "2"},  // match by transition name
		{"In Progress", "2"}, // match by target status
		{"in progress", "2"}, // case-insensitive
		{"START WORK", "2"},  // case-insensitive name
		{"To Do", "1"},       // status match
		{"Done", ""},         // no match
	}
	for _, c := range cases {
		got := PickTransition(ts, c.target)
		if c.wantID == "" {
			if got != nil {
				t.Errorf("PickTransition(%q) = %v, want nil", c.target, got)
			}
			continue
		}
		if got == nil || got.ID != c.wantID {
			t.Errorf("PickTransition(%q) = %v, want ID %s", c.target, got, c.wantID)
		}
	}

	// A transition whose NAME equals another transition's target status:
	// name match must win.
	ambiguous := []Transition{
		{ID: "1", Name: "In Progress", To: "In Progress"},
		{ID: "2", Name: "Start work", To: "In Progress"},
	}
	if got := PickTransition(ambiguous, "In Progress"); got == nil || got.ID != "1" {
		t.Errorf("name match should win, got %v", got)
	}
}

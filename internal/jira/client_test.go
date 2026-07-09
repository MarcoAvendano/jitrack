package jira

import "testing"

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

package gitlab

import "testing"

func TestParseRemoteURL(t *testing.T) {
	cases := []struct {
		in, owner, repo string
		wantErr         bool
	}{
		{"git@gitlab.com:my-org/my-repo.git", "my-org", "my-repo", false},
		{"https://gitlab.com/my-org/my-repo.git", "my-org", "my-repo", false},
		{"https://gitlab.com/my-org/my-repo", "my-org", "my-repo", false},
		{"ssh://git@gitlab.com/Owner/repo.git", "Owner", "repo", false},
		// nested subgroups: everything after the first segment is the "repo".
		{"git@gitlab.com:group/subgroup/repo.git", "group", "subgroup/repo", false},
		{"not-a-url", "", "", true},
	}
	for _, c := range cases {
		owner, repo, err := ParseRemoteURL(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("ParseRemoteURL(%q) error = %v, wantErr %v", c.in, err, c.wantErr)
			continue
		}
		if owner != c.owner || repo != c.repo {
			t.Errorf("ParseRemoteURL(%q) = %q/%q, want %q/%q", c.in, owner, repo, c.owner, c.repo)
		}
	}
}

func TestProjectPath(t *testing.T) {
	if got := projectPath("group", "repo"); got != "group%2Frepo" {
		t.Errorf("projectPath = %q, want group%%2Frepo", got)
	}
	if got := projectPath("group", "sub/repo"); got != "group%2Fsub%2Frepo" {
		t.Errorf("projectPath nested = %q", got)
	}
}

func TestMerged(t *testing.T) {
	if !(&MergeRequest{State: "merged"}).Merged() {
		t.Error("state=merged should be Merged")
	}
	if !(&MergeRequest{State: "closed", MergedAt: "2026-01-01T00:00:00Z"}).Merged() {
		t.Error("merged_at set should be Merged")
	}
	if (&MergeRequest{State: "closed"}).Merged() {
		t.Error("plain closed should not be Merged")
	}
}

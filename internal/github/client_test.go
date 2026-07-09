package github

import "testing"

func TestParseRemoteURL(t *testing.T) {
	cases := []struct {
		in, owner, repo string
		wantErr         bool
	}{
		{"git@github.com:Team-Storyrocket/storyrocket-react.git", "Team-Storyrocket", "storyrocket-react", false},
		{"https://github.com/Team-Storyrocket/storyrocket-react.git", "Team-Storyrocket", "storyrocket-react", false},
		{"https://github.com/Team-Storyrocket/storyrocket-react", "Team-Storyrocket", "storyrocket-react", false},
		{"ssh://git@github.com/Owner/repo.git", "Owner", "repo", false},
		{"git@github.mycorp.com:Owner/repo.git", "Owner", "repo", false},
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

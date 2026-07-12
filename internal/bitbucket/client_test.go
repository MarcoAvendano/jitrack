package bitbucket

import "testing"

func TestParseRemoteURL(t *testing.T) {
	cases := []struct {
		in, workspace, repo string
		wantErr             bool
	}{
		{"git@bitbucket.org:my-workspace/my-repo.git", "my-workspace", "my-repo", false},
		{"https://bitbucket.org/my-workspace/my-repo.git", "my-workspace", "my-repo", false},
		{"https://user@bitbucket.org/my-workspace/my-repo.git", "my-workspace", "my-repo", false},
		{"ssh://git@bitbucket.org/Workspace/repo.git", "Workspace", "repo", false},
		{"not-a-url", "", "", true},
	}
	for _, c := range cases {
		ws, repo, err := ParseRemoteURL(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("ParseRemoteURL(%q) error = %v, wantErr %v", c.in, err, c.wantErr)
			continue
		}
		if ws != c.workspace || repo != c.repo {
			t.Errorf("ParseRemoteURL(%q) = %q/%q, want %q/%q", c.in, ws, repo, c.workspace, c.repo)
		}
	}
}

func TestPRAccessors(t *testing.T) {
	pr := PullRequest{ID: 7, State: "MERGED"}
	pr.Links.HTML.Href = "https://bitbucket.org/ws/repo/pull-requests/7"
	pr.Source.Branch.Name = "feature/KAN-1"

	if !pr.Merged() {
		t.Error("MERGED state should be Merged")
	}
	if (&PullRequest{State: "DECLINED"}).Merged() {
		t.Error("DECLINED should not be Merged")
	}
	if pr.URL() != "https://bitbucket.org/ws/repo/pull-requests/7" {
		t.Errorf("URL() = %q", pr.URL())
	}
	if pr.HeadRef() != "feature/KAN-1" {
		t.Errorf("HeadRef() = %q", pr.HeadRef())
	}
}

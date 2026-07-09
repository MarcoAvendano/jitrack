package ticket

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"SR-123", "SR-123", false},
		{"sr-123", "SR-123", false},
		{" sr2-9 ", "SR2-9", false},
		{"123", "", true},
		{"SR-", "", true},
		{"S-123", "", true}, // project keys are at least two chars
		{"feature/SR-123", "", true},
		{"", "", true},
	}
	for _, c := range cases {
		got, err := Normalize(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("Normalize(%q) error = %v, wantErr %v", c.in, err, c.wantErr)
			continue
		}
		if got != c.want {
			t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExtractFromBranch(t *testing.T) {
	cases := []struct{ in, want string }{
		{"feature/SR-123-fix-login-redirect", "SR-123"},
		{"bugfix/SR-124", "SR-124"},
		{"sr-99-lowercase-branch", "SR-99"},
		{"main", ""},
		{"feature/no-ticket-here", ""},
	}
	for _, c := range cases {
		if got := ExtractFromBranch(c.in); got != c.want {
			t.Errorf("ExtractFromBranch(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Fix login redirect", "fix-login-redirect"},
		{"  [Bug] crash: on save!! ", "bug-crash-on-save"},
		{"Ünïcode & symbols #42", "n-code-symbols-42"},
		{"", ""},
		{"---", ""},
	}
	for _, c := range cases {
		if got := Slugify(c.in); got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	long := Slugify("this is a very long summary that keeps going and going and definitely exceeds the sixty character limit")
	if len(long) > maxSlugLen {
		t.Errorf("Slugify long input = %d chars, want <= %d", len(long), maxSlugLen)
	}
	if long[len(long)-1] == '-' {
		t.Errorf("Slugify long input ends with '-': %q", long)
	}
}

func TestBranch(t *testing.T) {
	cases := []struct {
		prefix, key, summary, want string
	}{
		{"feature", "SR-123", "Fix login redirect", "feature/SR-123-fix-login-redirect"},
		{"fix", "SR-124", "Crash on save", "fix/SR-124-crash-on-save"},
		{"chore", "SR-125", "", "chore/SR-125"},
		{"", "SR-1", "x", "feature/SR-1-x"},
	}
	for _, c := range cases {
		if got := Branch(c.prefix, c.key, c.summary); got != c.want {
			t.Errorf("Branch(%q, %q, %q) = %q, want %q", c.prefix, c.key, c.summary, got, c.want)
		}
	}
}

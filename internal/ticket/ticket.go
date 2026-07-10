// Package ticket holds the pure logic around Jira ticket keys and branch names.
package ticket

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const maxSlugLen = 60

var (
	keyRe        = regexp.MustCompile(`^[A-Z][A-Z0-9]+-[0-9]+$`)
	extractRe    = regexp.MustCompile(`[A-Z][A-Z0-9]+-[0-9]+`)
	nonAlnumRe   = regexp.MustCompile(`[^a-z0-9]+`)
	apostropheRe = regexp.MustCompile("['’`]")
)

// Normalize validates a user-supplied ticket ID, accepting lowercase input
// (sr-123 → SR-123). Returns an error if it doesn't look like a Jira key.
func Normalize(input string) (string, error) {
	key := strings.ToUpper(strings.TrimSpace(input))
	if !keyRe.MatchString(key) {
		return "", fmt.Errorf("%q does not look like a Jira ticket ID (expected e.g. SR-123)", input)
	}
	return key, nil
}

// ExtractFromBranch finds a ticket key inside a branch name such as
// "feature/SR-123-fix-login". Returns "" if none is present.
func ExtractFromBranch(branch string) string {
	return extractRe.FindString(strings.ToUpper(branch))
}

// Slugify turns an issue summary into a branch-safe slug: accents
// transliterated (é→e, ñ→n), apostrophes dropped ("don't"→"dont"),
// lowercase, remaining runs of non-alphanumerics collapsed to "-",
// trimmed, truncated at a word boundary to keep branch names readable.
func Slugify(summary string) string {
	slug := stripDiacritics(summary)
	slug = apostropheRe.ReplaceAllString(slug, "")
	slug = strings.ToLower(slug)
	slug = nonAlnumRe.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > maxSlugLen {
		slug = slug[:maxSlugLen]
		if i := strings.LastIndex(slug, "-"); i > 0 {
			slug = slug[:i]
		}
	}
	return slug
}

// stripDiacritics decomposes accented characters (NFD) and drops the
// combining marks, so é→e and ñ→n instead of becoming slug hyphens.
func stripDiacritics(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range norm.NFD.String(s) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Branch builds "<prefix>/<KEY>-<slug>". An empty summary yields just
// "<prefix>/<KEY>"; an empty prefix falls back to "feature".
func Branch(prefix, key, summary string) string {
	if prefix == "" {
		prefix = "feature"
	}
	name := prefix + "/" + key
	if slug := Slugify(summary); slug != "" {
		name += "-" + slug
	}
	return name
}

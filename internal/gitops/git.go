// Package gitops wraps the git CLI. Shelling out (rather than a Go git
// library) keeps the user's SSH auth, hooks, and global config in play.
package gitops

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// run executes git with args and returns trimmed stdout.
func run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text != "" {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), text)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return text, nil
}

// runInteractive executes git attached to the terminal (for push output,
// commit hooks, credential prompts).
func runInteractive(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RepoRoot returns the repository top-level directory, or an error if the
// current directory is not inside a git repository.
func RepoRoot() (string, error) {
	root, err := run("rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not inside a git repository")
	}
	return root, nil
}

// CurrentBranch returns the checked-out branch name.
func CurrentBranch() (string, error) {
	return run("rev-parse", "--abbrev-ref", "HEAD")
}

// HasTrackedChanges reports whether tracked files have uncommitted changes
// (staged or not). Untracked files are ignored — they don't block a checkout.
func HasTrackedChanges() (bool, error) {
	out, err := run("status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

// HasStagedChanges reports whether anything is staged for commit.
func HasStagedChanges() (bool, error) {
	err := exec.Command("git", "diff", "--cached", "--quiet").Run()
	if err == nil {
		return false, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return true, nil
	}
	return false, fmt.Errorf("git diff --cached: %w", err)
}

// StatusSummary returns `git status --short` output for hints.
func StatusSummary() string {
	out, _ := run("status", "--short")
	return out
}

// LocalBranchExists reports whether a local branch with this name exists.
func LocalBranchExists(name string) bool {
	err := exec.Command("git", "rev-parse", "--verify", "--quiet", "refs/heads/"+name).Run()
	return err == nil
}

// Fetch updates refs from origin.
func Fetch() error {
	_, err := run("fetch", "origin")
	return err
}

// CreateBranch creates and checks out a branch from startPoint
// (e.g. origin/main).
func CreateBranch(name, startPoint string) error {
	_, err := run("checkout", "-b", name, startPoint, "--no-track")
	return err
}

// Checkout switches to an existing branch.
func Checkout(name string) error {
	_, err := run("checkout", name)
	return err
}

// Commit records staged changes with the given message (runs hooks,
// output attached to the terminal).
func Commit(message string) error {
	return runInteractive("commit", "-m", message)
}

// Push pushes the current branch to origin, setting upstream.
func Push() error {
	return runInteractive("push", "-u", "origin", "HEAD")
}

// CommitsAhead returns how many commits HEAD has that ref does not
// (git rev-list --count ref..HEAD).
func CommitsAhead(ref string) (int, error) {
	out, err := run("rev-list", "--count", ref+"..HEAD")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}

// BranchPushed reports whether the current branch has an upstream on origin.
func BranchPushed() bool {
	_, err := run("rev-parse", "--verify", "--quiet", "@{upstream}")
	return err == nil
}

// RemoteURL returns the origin remote URL.
func RemoteURL() (string, error) {
	url, err := run("remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("no 'origin' remote configured: %w", err)
	}
	return url, nil
}

// HeadSubject returns the last commit's short description, for summaries.
func HeadSubject() string {
	out, _ := run("log", "-1", "--pretty=%h %s")
	return out
}

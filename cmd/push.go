package cmd

import (
	"fmt"
	"strings"

	"github.com/MarcoAvendano/jitrack/internal/forge"
	"github.com/MarcoAvendano/jitrack/internal/gitops"
	"github.com/MarcoAvendano/jitrack/internal/jira"
	"github.com/MarcoAvendano/jitrack/internal/ticket"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var pushBaseFlag string

var pushCmd = &cobra.Command{
	Use:   "push [TICKET-ID]",
	Short: "Commit staged changes, push the branch, and open (or reuse) a pull request",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := gitops.RepoRoot(); err != nil {
			return err
		}
		branch, err := gitops.CurrentBranch()
		if err != nil {
			return err
		}

		var key string
		if len(args) == 1 {
			key, err = ticket.Normalize(args[0])
			if err != nil {
				return err
			}
		} else if key = ticket.ExtractFromBranch(branch); key == "" {
			return fmt.Errorf("no ticket ID given and none found in branch %q — run `jitrack push TICKET-123`", branch)
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		base := baseBranch(cfg, pushBaseFlag)

		staged, err := gitops.HasStagedChanges()
		if err != nil {
			return err
		}
		if !staged && !gitops.BranchPushed() {
			// No commit to make and no upstream yet — only proceed if the
			// branch has commits of its own to push (committed by hand).
			ahead, err := gitops.CommitsAhead("origin/" + base)
			if err == nil && ahead == 0 {
				hint := gitops.StatusSummary()
				if hint == "" {
					return fmt.Errorf("nothing staged and no commits to push — make some changes first")
				}
				return fmt.Errorf("nothing staged and no commits to push — stage what you want with `git add`, then retry:\n%s", hint)
			}
		}

		if err := cfg.RequireJira(); err != nil {
			return err
		}
		if err := cfg.RequireProvider(); err != nil {
			return err
		}

		// Also used for the PR title; in the no-commit path it stays as
		// inferred from the branch prefix.
		ctype := commitTypeFromBranch(branch)

		if staged {
			err = huh.NewSelect[string]().
				Title(fmt.Sprintf("Commit type for %s", key)).
				Options(huh.NewOptions("feat", "fix", "hotfix", "chore")...).
				Value(&ctype).Run()
			if err != nil {
				return err
			}

			var message string
			err = huh.NewInput().
				Title(fmt.Sprintf("Commit message (will be \"%s: %s <message>\")", ctype, key)).
				Validate(requireNonEmpty("commit message")).
				Value(&message).Run()
			if err != nil {
				return err
			}
			message = strings.TrimSpace(message)
			// Drop a hand-typed leading ticket key — the format adds it.
			if up := strings.ToUpper(message); strings.HasPrefix(up, key) {
				message = strings.TrimSpace(message[len(key):])
				message = strings.TrimSpace(strings.TrimPrefix(message, ":"))
			}
			message = fmt.Sprintf("%s: %s %s", ctype, key, message)

			if err := gitops.Commit(message); err != nil {
				return err
			}
			fmt.Printf("✔ committed: %s\n", gitops.HeadSubject())
		} else {
			// Commits already exist (made by hand) or the branch was pushed
			// before: skip the commit step, sync the branch, ensure the PR.
			fmt.Println("nothing staged — pushing existing commits and ensuring PR exists")
		}
		if err := gitops.Push(); err != nil {
			return err
		}
		fmt.Printf("✔ pushed %s to origin\n", branch)

		remote, _ := gitops.RemoteURL()
		fp, err := forge.New(cfg, remote)
		if err != nil {
			return err
		}

		pr, err := fp.FindOpenPR(branch)
		if err != nil {
			return err
		}
		jc := jira.NewClient(cfg.Get("jira.url"), cfg.Get("jira.email"), cfg.Get("jira.token"))
		if pr != nil {
			fmt.Printf("✔ pull request already open: %s\n", pr.URL)
			return nil
		}

		title := fmt.Sprintf("%s: %s", ctype, key)
		if issue, err := jc.GetIssue(key); err == nil && issue.Summary != "" {
			title = fmt.Sprintf("%s: %s %s", ctype, key, issue.Summary)
		}
		body := fmt.Sprintf("Jira ticket: [%s](%s)", key, jc.BrowseURL(key))
		pr, err = fp.CreatePR(title, branch, base, body)
		if err != nil {
			return fmt.Errorf("creating pull request (%s → %s): %w", branch, base, err)
		}
		fmt.Printf("✔ pull request created: %s\n", pr.URL)

		if err := jc.AddComment(key, fmt.Sprintf("Pull request created: %s", pr.URL)); err != nil {
			fmt.Printf("⚠ could not comment PR link on %s: %v\n", key, err)
		} else {
			fmt.Printf("✔ commented PR link on %s\n", key)
		}
		return nil
	},
}

// commitTypeFromBranch preselects the commit type from the branch prefix
// (feature/KAN-123-… → feat). Unknown or missing prefixes default to feat.
func commitTypeFromBranch(branch string) string {
	prefix, _, _ := strings.Cut(branch, "/")
	switch prefix {
	case "feature", "feat":
		return "feat"
	case "fix", "bugfix":
		return "fix"
	case "hotfix":
		return "hotfix"
	case "chore":
		return "chore"
	}
	return "feat"
}

func init() {
	pushCmd.Flags().StringVar(&pushBaseFlag, "base", "", "branch the pull request targets (defaults to base_branch from config)")
	rootCmd.AddCommand(pushCmd)
}

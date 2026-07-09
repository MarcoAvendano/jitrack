package cmd

import (
	"fmt"
	"strings"

	"github.com/MarcoAvendano/cli-jira-git-workflow/internal/github"
	"github.com/MarcoAvendano/cli-jira-git-workflow/internal/gitops"
	"github.com/MarcoAvendano/cli-jira-git-workflow/internal/jira"
	"github.com/MarcoAvendano/cli-jira-git-workflow/internal/ticket"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

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
			return fmt.Errorf("no ticket ID given and none found in branch %q — run `sr-cli push TICKET-123`", branch)
		}

		staged, err := gitops.HasStagedChanges()
		if err != nil {
			return err
		}
		if !staged && !gitops.BranchPushed() {
			// Nothing to commit and the branch was never pushed: there is
			// nothing to make a PR from either.
			hint := gitops.StatusSummary()
			if hint == "" {
				return fmt.Errorf("nothing staged and no local changes — make some changes first")
			}
			return fmt.Errorf("nothing staged for commit — stage what you want with `git add`, then retry:\n%s", hint)
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if err := cfg.RequireJira(); err != nil {
			return err
		}
		if err := cfg.RequireGitHub(); err != nil {
			return err
		}

		if staged {
			var message string
			err = huh.NewInput().
				Title(fmt.Sprintf("Commit message (will be prefixed \"%s: \")", key)).
				Validate(requireNonEmpty("commit message")).
				Value(&message).Run()
			if err != nil {
				return err
			}
			message = strings.TrimSpace(message)
			if !strings.HasPrefix(strings.ToUpper(message), key+":") {
				message = key + ": " + message
			}

			if err := gitops.Commit(message); err != nil {
				return err
			}
			fmt.Printf("✔ committed: %s\n", gitops.HeadSubject())
		} else {
			// Branch already pushed once: resume — sync it and make sure
			// the PR exists (e.g. a previous run failed at the PR step).
			fmt.Println("nothing staged — ensuring branch is pushed and PR exists")
		}
		if err := gitops.Push(); err != nil {
			return err
		}
		fmt.Printf("✔ pushed %s to origin\n", branch)

		owner, repo, err := resolveRepo(cfg)
		if err != nil {
			return err
		}
		gh := github.NewClient(cfg.Get("github.api_url"), cfg.Get("github.token"))

		pr, err := gh.FindOpenPR(owner, repo, branch)
		if err != nil {
			return err
		}
		jc := jira.NewClient(cfg.Get("jira.url"), cfg.Get("jira.email"), cfg.Get("jira.token"))
		if pr != nil {
			fmt.Printf("✔ pull request already open: %s\n", pr.HTMLURL)
			return nil
		}

		title := key
		if issue, err := jc.GetIssue(key); err == nil && issue.Summary != "" {
			title = fmt.Sprintf("%s: %s", key, issue.Summary)
		}
		base := cfg.Get("base_branch")
		body := fmt.Sprintf("Jira ticket: [%s](%s)", key, jc.BrowseURL(key))
		pr, err = gh.CreatePR(owner, repo, title, branch, base, body)
		if err != nil {
			return fmt.Errorf("creating pull request (%s → %s): %w", branch, base, err)
		}
		fmt.Printf("✔ pull request created: %s\n", pr.HTMLURL)

		if err := jc.AddComment(key, fmt.Sprintf("Pull request created: %s", pr.HTMLURL)); err != nil {
			fmt.Printf("⚠ could not comment PR link on %s: %v\n", key, err)
		} else {
			fmt.Printf("✔ commented PR link on %s\n", key)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}

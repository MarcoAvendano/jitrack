package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/MarcoAvendano/jitrack/internal/gitops"
	"github.com/MarcoAvendano/jitrack/internal/jira"
	"github.com/MarcoAvendano/jitrack/internal/ticket"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start TICKET-ID",
	Short: "Start work on a ticket: branch off base, move it to In Progress, comment the branch name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := ticket.Normalize(args[0])
		if err != nil {
			return err
		}
		if _, err := gitops.RepoRoot(); err != nil {
			return err
		}
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if err := cfg.RequireJira(); err != nil {
			return err
		}
		jc := jira.NewClient(cfg.Get("jira.url"), cfg.Get("jira.email"), cfg.Get("jira.token"))

		fmt.Printf("Fetching %s from Jira… ", key)
		issue, err := jc.GetIssue(key)
		if err != nil {
			return err
		}
		fmt.Printf("✔ [%s] %s (%s)\n", issue.IssueType, issue.Summary, issue.Status)

		// Let the user pick the prefix; the issue-type mapping from config
		// picks which option is preselected (e.g. Bug → fix).
		prefix := cfg.BranchPrefix(issue.IssueType)
		options := []string{"feature", "fix", "hotfix", "chore"}
		if !slices.Contains(options, prefix) && prefix != "" {
			options = append([]string{prefix}, options...)
		}
		err = huh.NewSelect[string]().
			Title(fmt.Sprintf("Branch prefix for %s (%s)", key, issue.IssueType)).
			Options(huh.NewOptions(options...)...).
			Value(&prefix).Run()
		if err != nil {
			return err
		}

		branch := ticket.Branch(prefix, issue.Key, issue.Summary)

		if gitops.LocalBranchExists(branch) {
			checkout := true
			err := huh.NewConfirm().
				Title(fmt.Sprintf("Branch %s already exists. Check it out?", branch)).
				Value(&checkout).Run()
			if err != nil {
				return err
			}
			if !checkout {
				return fmt.Errorf("aborted")
			}
			if err := gitops.Checkout(branch); err != nil {
				return err
			}
			fmt.Printf("✔ switched to existing branch %s\n", branch)
			transitionIssue(jc, issue, cfg.Get("transitions.start"))
			return nil
		}

		dirty, err := gitops.HasTrackedChanges()
		if err != nil {
			return err
		}
		if dirty {
			return fmt.Errorf("working tree has uncommitted changes — commit or stash them first:\n%s", gitops.StatusSummary())
		}

		base := cfg.Get("base_branch")
		fmt.Printf("Fetching origin… ")
		if err := gitops.Fetch(); err != nil {
			return err
		}
		fmt.Println("✔")
		if err := gitops.CreateBranch(branch, "origin/"+base); err != nil {
			return err
		}
		fmt.Printf("✔ created branch %s from origin/%s\n", branch, base)

		// The branch exists now; Jira hiccups shouldn't fail the command.
		transitionIssue(jc, issue, cfg.Get("transitions.start"))
		if err := jc.AddComment(key, fmt.Sprintf("Branch created: %s", branch)); err != nil {
			fmt.Printf("⚠ could not comment on ticket: %v\n", err)
		} else {
			fmt.Printf("✔ commented branch name on %s\n", key)
		}

		fmt.Printf("\nReady to work on %s — %s\n", key, jc.BrowseURL(key))
		return nil
	},
}

// transitionIssue moves the issue to the given transition/status, skipping
// when it's already there. Failures warn instead of aborting — the git work
// around it has already happened.
func transitionIssue(jc *jira.Client, issue *jira.Issue, target string) {
	if strings.EqualFold(issue.Status, target) {
		fmt.Printf("✔ %s already in %s\n", issue.Key, issue.Status)
		return
	}
	if err := jc.TransitionTo(issue.Key, target); err != nil {
		fmt.Printf("⚠ could not move ticket to %q: %v\n", target, err)
	} else {
		fmt.Printf("✔ %s moved to %s\n", issue.Key, target)
	}
}

func init() {
	rootCmd.AddCommand(startCmd)
}

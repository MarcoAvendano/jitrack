package cmd

import (
	"fmt"

	"github.com/MarcoAvendano/jitrack/internal/forge"
	"github.com/MarcoAvendano/jitrack/internal/gitops"
	"github.com/MarcoAvendano/jitrack/internal/jira"
	"github.com/MarcoAvendano/jitrack/internal/ticket"
	"github.com/spf13/cobra"
)

var closeBaseFlag string

var closeCmd = &cobra.Command{
	Use:   "close [TICKET-ID]",
	Short: "After the ticket's PR is closed: move the ticket onward (e.g. Ready to QA) and switch back to the base branch",
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
			return fmt.Errorf("no ticket ID given and none found in branch %q — run `jitrack close TICKET-123`", branch)
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if err := cfg.RequireJira(); err != nil {
			return err
		}
		if err := cfg.RequireProvider(); err != nil {
			return err
		}

		remote, _ := gitops.RemoteURL()
		fp, err := forge.New(cfg, remote)
		if err != nil {
			return err
		}

		// Find the ticket's PR by looking for its key in PR head branches.
		prs, err := fp.ListPRs("all")
		if err != nil {
			return err
		}
		var pr *forge.PullRequest
		for i := range prs {
			if ticket.ExtractFromBranch(prs[i].HeadRef) == key {
				pr = &prs[i] // newest first — the current attempt wins
				break
			}
		}
		if pr == nil {
			return fmt.Errorf("no pull request found for %s — nothing to close", key)
		}
		if pr.State == "open" {
			return fmt.Errorf("PR #%d for %s is still open — merge or close it first: %s", pr.Number, key, pr.URL)
		}
		if pr.Merged {
			fmt.Printf("✔ PR #%d merged: %s\n", pr.Number, pr.URL)
		} else {
			fmt.Printf("⚠ PR #%d was closed without merging: %s\n", pr.Number, pr.URL)
		}

		jc := jira.NewClient(cfg.Get("jira.url"), cfg.Get("jira.email"), cfg.Get("jira.token"))
		issue, err := jc.GetIssue(key)
		if err != nil {
			return err
		}
		transitionIssue(jc, issue, cfg.Get("transitions.close"))

		base := baseBranch(cfg, closeBaseFlag)
		if branch == base {
			fmt.Printf("✔ already on %s\n", base)
			return nil
		}
		if err := gitops.Checkout(base); err != nil {
			return err
		}
		fmt.Printf("✔ switched to %s\n", base)
		return nil
	},
}

func init() {
	closeCmd.Flags().StringVar(&closeBaseFlag, "base", "", "branch to switch back to (defaults to base_branch from config)")
	rootCmd.AddCommand(closeCmd)
}

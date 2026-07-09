package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:           "jitrack",
	Short:         "Sync your git workflow with Jira tickets",
	Long:          "jitrack automates the Jira ↔ git workflow: start a ticket (branch + In Progress + comment) and push work (commit + push + pull request).",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: false,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

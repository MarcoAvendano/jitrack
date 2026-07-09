package cmd

import (
	"fmt"
	"strings"

	"github.com/MarcoAvendano/jitrack/internal/config"
	"github.com/MarcoAvendano/jitrack/internal/github"
	"github.com/MarcoAvendano/jitrack/internal/jira"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive setup: configure and validate the Jira and GitHub connections",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		existing, err := loadConfig()
		if err != nil {
			return err
		}
		jiraURL := existing.Get("jira.url")
		jiraEmail := existing.Get("jira.email")
		jiraToken := ""
		ghToken := ""

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Jira URL").
					Description("Your Jira Cloud base URL, e.g. https://yourteam.atlassian.net").
					Placeholder("https://yourteam.atlassian.net").
					Value(&jiraURL).
					Validate(requireNonEmpty("Jira URL")),
				huh.NewInput().
					Title("Jira email").
					Description("The Atlassian account email the API token belongs to").
					Value(&jiraEmail).
					Validate(requireNonEmpty("Jira email")),
				huh.NewInput().
					Title("Jira API token").
					Description("Create one at https://id.atlassian.com/manage-profile/security/api-tokens").
					EchoMode(huh.EchoModePassword).
					Value(&jiraToken).
					Validate(requireNonEmpty("Jira API token")),
				huh.NewInput().
					Title("GitHub token").
					Description("Fine-grained PAT with Pull requests read/write — https://github.com/settings/tokens").
					EchoMode(huh.EchoModePassword).
					Value(&ghToken).
					Validate(requireNonEmpty("GitHub token")),
			),
		)
		if err := form.Run(); err != nil {
			return err
		}
		jiraURL = normalizeURL(jiraURL)

		fmt.Print("Validating Jira credentials… ")
		name, err := jira.NewClient(jiraURL, jiraEmail, jiraToken).Myself()
		if err != nil {
			return fmt.Errorf("\n%w", err)
		}
		fmt.Printf("✔ authenticated as %s\n", name)

		fmt.Print("Validating GitHub credentials… ")
		login, err := github.NewClient(existing.Get("github.api_url"), ghToken).Viewer()
		if err != nil {
			return fmt.Errorf("\n%w", err)
		}
		fmt.Printf("✔ authenticated as %s\n", login)

		path, err := config.GlobalPath()
		if err != nil {
			return err
		}
		for key, value := range map[string]string{
			"jira.url":     jiraURL,
			"jira.email":   jiraEmail,
			"jira.token":   jiraToken,
			"github.token": ghToken,
		} {
			if err := config.Set(path, key, value); err != nil {
				return err
			}
		}
		fmt.Printf("✔ config saved to %s\n", path)
		fmt.Println("\nYou're set. Try: sr-cli start TICKET-123")
		return nil
	},
}

func requireNonEmpty(field string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s is required", field)
		}
		return nil
	}
}

func normalizeURL(u string) string {
	u = strings.TrimRight(strings.TrimSpace(u), "/")
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}
	return u
}

func init() {
	rootCmd.AddCommand(initCmd)
}

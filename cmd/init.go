package cmd

import (
	"fmt"
	"strings"

	"github.com/MarcoAvendano/jitrack/internal/config"
	"github.com/MarcoAvendano/jitrack/internal/forge"
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
		provider := existing.Provider()
		providerToken := ""
		bitbucketUser := existing.Get("bitbucket.username")

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Git provider").
					Description("Where your pull/merge requests live").
					Options(huh.NewOptions(forge.Providers()...)...).
					Value(&provider),
			),
			// Bitbucket authenticates with Basic auth, so it also needs a
			// username. Hidden for the token-only providers.
			huh.NewGroup(
				huh.NewInput().
					Title("Bitbucket username").
					Description("Your Bitbucket username, or Atlassian email if using an API token").
					Value(&bitbucketUser).
					Validate(requireNonEmpty("Bitbucket username")),
			).WithHideFunc(func() bool { return provider != "bitbucket" }),
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
					TitleFunc(func() string { return providerTokenTitle(provider) }, &provider).
					DescriptionFunc(func() string { return providerTokenHelp(provider) }, &provider).
					EchoMode(huh.EchoModePassword).
					Value(&providerToken).
					Validate(requireNonEmpty("access token")),
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

		fmt.Printf("Validating %s credentials… ", provider)
		login, err := forge.Validate(provider, existing.Get(provider+".api_url"), bitbucketUser, providerToken)
		if err != nil {
			return fmt.Errorf("\n%w", err)
		}
		fmt.Printf("✔ authenticated as %s\n", login)

		path, err := config.GlobalPath()
		if err != nil {
			return err
		}
		values := map[string]string{
			"provider":          provider,
			"jira.url":          jiraURL,
			"jira.email":        jiraEmail,
			"jira.token":        jiraToken,
			provider + ".token": providerToken,
		}
		if provider == "bitbucket" {
			values["bitbucket.username"] = bitbucketUser
		}
		for key, value := range values {
			if err := config.Set(path, key, value); err != nil {
				return err
			}
		}
		fmt.Printf("✔ config saved to %s\n", path)
		fmt.Println("\nYou're set. Try: jitrack start TICKET-123")
		return nil
	},
}

// providerTokenTitle / providerTokenHelp render the token prompt for the
// selected provider (the field updates live as the provider changes).
func providerTokenTitle(provider string) string {
	switch provider {
	case "gitlab":
		return "GitLab token"
	case "bitbucket":
		return "Bitbucket app password / API token"
	default:
		return "GitHub token"
	}
}

func providerTokenHelp(provider string) string {
	switch provider {
	case "gitlab":
		return "Personal access token with the 'api' scope — https://gitlab.com/-/user_settings/personal_access_tokens"
	case "bitbucket":
		return "App password (Pull requests: Read/Write) — https://bitbucket.org/account/settings/app-passwords/"
	default:
		return "Fine-grained PAT with Pull requests read/write + Contents read — https://github.com/settings/tokens"
	}
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

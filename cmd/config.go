package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/MarcoAvendano/cli-jira-git-workflow/internal/config"
	"github.com/MarcoAvendano/cli-jira-git-workflow/internal/gitops"
	"github.com/spf13/cobra"
)

var configRepoFlag bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage sr-cli configuration (JSON files, editable by hand too)",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value (e.g. jira.url, github.token, base_branch)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]
		path, err := configWritePath()
		if err != nil {
			return err
		}
		if err := config.Set(path, key, value); err != nil {
			return err
		}
		fmt.Printf("✔ %s set in %s\n", key, path)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Print one effective config value (tokens masked)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		if !config.ValidKey(key) {
			return fmt.Errorf("unknown config key %q", key)
		}
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		fmt.Println(config.Mask(key, cfg.Get(key)))
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Print the merged effective config and where each value comes from",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		for _, key := range cfg.Keys() {
			fmt.Printf("%-30s %-20s (%s)\n", key, config.Mask(key, cfg.Get(key)), cfg.Source(key))
		}
		return nil
	},
}

func configWritePath() (string, error) {
	if configRepoFlag {
		root, err := gitops.RepoRoot()
		if err != nil {
			return "", fmt.Errorf("--repo requires running inside a git repository")
		}
		return filepath.Join(root, config.RepoFileName), nil
	}
	return config.GlobalPath()
}

// loadConfig merges global + repo (when inside one) + env.
func loadConfig() (*config.Config, error) {
	repoDir, err := gitops.RepoRoot()
	if err != nil {
		repoDir = "" // outside a repo: global + env only
	}
	return config.Load(repoDir)
}

func init() {
	configSetCmd.Flags().BoolVar(&configRepoFlag, "repo", false, "write to the repo's .sr-cli.json instead of the global config")
	configCmd.AddCommand(configSetCmd, configGetCmd, configListCmd)
	rootCmd.AddCommand(configCmd)
}

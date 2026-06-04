package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/khuedoan/cloudlab/toolbox/internal/secrets"
)

var settingsFile string

func init() {
	secretsCmd.Flags().StringVar(&settingsFile, "settings", "", "Path to settings YAML file")
	_ = secretsCmd.MarkFlagRequired("settings")
}

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage secrets in Vault",
	PreRunE: func(_ *cobra.Command, _ []string) error {
		return requireExecutables("kubectl")
	},
	RunE: runSecrets,
}

func runSecrets(cmd *cobra.Command, _ []string) error {
	config, err := secrets.LoadConfig(settingsFile)
	if err != nil {
		return fmt.Errorf("load settings file: %w", err)
	}

	entries, err := secrets.ParseAndValidate(config)
	if err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	vault, stopVault, err := connectVault(cmd.Context())
	if err != nil {
		return fmt.Errorf("connect to Vault: %w", err)
	}
	defer stopVault()
	log.Debug("connected to Vault")

	service := secrets.NewService(vault, secrets.HuhPrompter{})
	if err := service.Run(cmd.Context(), entries); err != nil {
		return err
	}

	log.Info("all secrets processed successfully")
	return nil
}

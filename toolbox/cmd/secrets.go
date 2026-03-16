package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/khuedoan/cloudlab/toolbox/internal/cluster"
	"github.com/khuedoan/cloudlab/toolbox/internal/secrets"
)

const connectTimeout = 30 * time.Second

var settingsFile string

func init() {
	secretsCmd.Flags().StringVar(&settingsFile, "settings", "", "Path to settings YAML file")
	secretsCmd.MarkFlagRequired("settings")
}

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage secrets in Vault",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return validateClusterFlags()
	},
	RunE: runSecrets,
}

func runSecrets(cmd *cobra.Command, args []string) error {
	config, err := secrets.LoadConfig(settingsFile)
	if err != nil {
		return fmt.Errorf("load settings file: %w", err)
	}

	entries, err := secrets.ParseAndValidate(config)
	if err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	connectCtx, cancel := context.WithTimeout(cmd.Context(), connectTimeout)
	defer cancel()

	client, err := cluster.NewClient(connectCtx, cluster.ClientConfig{
		HostsFile:     hostsFile,
		Host:          host,
		SSHUser:       sshUser,
		SSHKey:        sshKey,
		SSHKnownHosts: sshKnownHosts,
	})
	if err != nil {
		return fmt.Errorf("connect to cluster: %w", err)
	}
	defer client.Close()
	log.Debug("connected to cluster")

	service := secrets.NewService(client.Vault(), secrets.HuhPrompter{})
	if err := service.Run(cmd.Context(), entries); err != nil {
		return err
	}

	log.Info("all secrets processed successfully")
	return nil
}

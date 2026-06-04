package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

const (
	podDetachTimeout = 5 * time.Minute
	restoreTimeout   = 30 * time.Minute
)

var (
	backupEnv             string
	backupSettingsFile    string
	backupVolumeSelectors []string
)

func init() {
	backupCmd.PersistentFlags().StringVar(&backupEnv, "env", "", "Environment to manage backups for")
	backupCmd.PersistentFlags().StringVar(&backupSettingsFile, "settings", "settings.yaml", "Path to settings YAML file")
	backupCmd.PersistentFlags().StringArrayVar(&backupVolumeSelectors, "volume", nil, "Configured volume to operate on in namespace/pvc format; repeat for multiple volumes")

	backupCmd.AddCommand(backupSetupCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	rootCmd.AddCommand(backupCmd)
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create VolSync backup and restore resources",
}

var backupSetupCmd = &cobra.Command{
	Use:   "setup",
	Args:  cobra.NoArgs,
	Short: "Create or patch VolSync ReplicationSource resources",
	PreRunE: func(_ *cobra.Command, _ []string) error {
		return validateBackupFlags()
	},
	RunE: runBackupSetup,
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore",
	Args:  cobra.NoArgs,
	Short: "Create or patch VolSync ReplicationDestination resources",
	PreRunE: func(_ *cobra.Command, _ []string) error {
		return validateBackupFlags()
	},
	RunE: runBackupRestore,
}

func validateBackupFlags() error {
	if backupEnv == "" {
		return fmt.Errorf("--env is required")
	}
	return requireExecutables("kubectl")
}

package cmd

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	hostsFile string
	host      string
	sshUser   string
	sshKey    string
)

func init() {
	log.SetReportTimestamp(false)

	rootCmd.PersistentFlags().StringVar(&hostsFile, "hosts-file", "", "Path to hosts.json file")
	rootCmd.PersistentFlags().StringVar(&host, "host", "", "Host name to connect to (e.g., kube-1)")
	rootCmd.PersistentFlags().StringVar(&sshUser, "ssh-user", "root", "SSH user")
	rootCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", defaultSSHKey(), "Path to SSH private key")

	rootCmd.MarkPersistentFlagRequired("hosts-file")
	rootCmd.MarkPersistentFlagRequired("host")

	rootCmd.AddCommand(secretsCmd)
}

var rootCmd = &cobra.Command{
	Use:   "toolbox",
	Short: "CLI tools for managing cloudlab infrastructure",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func defaultSSHKey() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "id_ed25519")
}

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	hostsFile     string
	host          string
	sshUser       string
	sshKey        string
	sshKnownHosts string
)

func init() {
	log.SetReportTimestamp(false)

	rootCmd.PersistentFlags().StringVar(&hostsFile, "hosts-file", "", "Path to hosts.json file")
	rootCmd.PersistentFlags().StringVar(&host, "host", "", "Host name to connect to (e.g., kube-1)")
	rootCmd.PersistentFlags().StringVar(&sshUser, "ssh-user", "root", "SSH user")
	rootCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", defaultSSHKey(), "Path to SSH private key")
	rootCmd.PersistentFlags().StringVar(&sshKnownHosts, "ssh-known-hosts", defaultKnownHostsFile(), "Path to SSH known_hosts file")

	rootCmd.AddCommand(gitopsCmd)
	rootCmd.AddCommand(secretsCmd)
	rootCmd.AddCommand(vendorCmd)
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

func defaultKnownHostsFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "known_hosts")
}

func validateClusterFlags() error {
	if hostsFile == "" {
		return fmt.Errorf("--hosts-file is required")
	}
	if host == "" {
		return fmt.Errorf("--host is required")
	}
	return nil
}

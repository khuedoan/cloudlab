package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

func init() {
	log.SetReportTimestamp(false)

	rootCmd.AddCommand(gitopsCmd)
	rootCmd.AddCommand(secretsCmd)
	rootCmd.AddCommand(vendorCmd)
}

var rootCmd = &cobra.Command{
	Use:   "toolbox",
	Short: "CLI tools for managing cloudlab infrastructure",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func requireExecutables(names ...string) error {
	for _, name := range names {
		if _, err := exec.LookPath(name); err != nil {
			return fmt.Errorf("find %s CLI: %w", name, err)
		}
	}
	return nil
}

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/khuedoan/cloudlab/toolbox/internal/vendors"
)

var vendorCmd = &cobra.Command{
	Use:   "vendor",
	Args:  cobra.NoArgs,
	Short: "Vendor charts and images from settings.yaml into the in-cluster registry",
	PreRunE: func(_ *cobra.Command, _ []string) error {
		return requireExecutables("helm", "kubectl", "oras")
	},
	RunE: runSync,
}

func init() {
	vendorCmd.Flags().StringVar(&settingsFile, "settings", "", "Path to settings YAML file")
	_ = vendorCmd.MarkFlagRequired("settings")
}

func runSync(cmd *cobra.Command, _ []string) error {
	entries, err := vendors.LoadVendors(settingsFile)
	if err != nil {
		return err
	}

	tunnel, err := startKubectlPortForward(cmd.Context(), registryNamespace, registryService, registryPort)
	if err != nil {
		return fmt.Errorf("forward registry: %w", err)
	}
	defer tunnel.Close()

	workdir, err := os.MkdirTemp("", "toolbox-vendor-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(workdir)

	return vendors.Sync(cmd.Context(), workdir, tunnel.addr, entries)
}

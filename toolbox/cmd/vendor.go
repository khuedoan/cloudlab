package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/khuedoan/cloudlab/toolbox/internal/cluster"
	"github.com/khuedoan/cloudlab/toolbox/internal/vendors"
)

var vendorCmd = &cobra.Command{
	Use:   "vendor",
	Args:  cobra.NoArgs,
	Short: "Vendor charts and images from settings.yaml into the in-cluster registry",
	PreRunE: func(_ *cobra.Command, _ []string) error {
		if err := validateClusterFlags(); err != nil {
			return err
		}
		if settingsFile == "" {
			return fmt.Errorf("--settings is required")
		}
		for _, name := range []string{"helm", "oras"} {
			if _, err := exec.LookPath(name); err != nil {
				return fmt.Errorf("find %s CLI: %w", name, err)
			}
		}
		return nil
	},
	RunE: runSync,
}

func init() {
	vendorCmd.Flags().StringVar(&settingsFile, "settings", "", "Path to settings YAML file")
}

func runSync(cmd *cobra.Command, _ []string) error {
	entries, err := vendors.LoadVendors(settingsFile)
	if err != nil {
		return err
	}

	connectCtx, cancel := context.WithTimeout(cmd.Context(), connectTimeout)
	defer cancel()

	hostAddr, err := cluster.LoadHost(hostsFile, host)
	if err != nil {
		return fmt.Errorf("load host: %w", err)
	}

	conn, err := cluster.Connect(cluster.SSHConfig{
		Host:           hostAddr,
		User:           sshUser,
		KeyPath:        sshKey,
		KnownHostsPath: sshKnownHosts,
		Timeout:        connectTimeout,
	})
	if err != nil {
		return fmt.Errorf("connect to cluster: %w", err)
	}
	defer conn.Close()

	tunnel, err := conn.Forward(connectCtx, cluster.ServiceConfig{
		Namespace: registryNamespace,
		Name:      registryService,
		Port:      registryPort,
	})
	if err != nil {
		return fmt.Errorf("forward registry: %w", err)
	}

	workdir, err := os.MkdirTemp("", "toolbox-vendor-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(workdir)

	return vendors.Sync(cmd.Context(), workdir, tunnel.LocalAddr, entries)
}

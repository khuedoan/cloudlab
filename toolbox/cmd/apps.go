package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	appbundle "github.com/khuedoan/cloudlab/toolbox/internal/apps"
)

const (
	appsRepository = "apps"
	appsTag        = "latest"
)

var appsPath string

func init() {
	appsCmd.Flags().StringVar(&appsPath, "path", "apps", "Path to the app manifest tree")
}

var appsCmd = &cobra.Command{
	Use:   "apps",
	Args:  cobra.NoArgs,
	Short: "Generate and push the apps manifest bundle",
	PreRunE: func(_ *cobra.Command, _ []string) error {
		return requireExecutables("flux", "kubectl")
	},
	RunE: runApps,
}

func runApps(cmd *cobra.Command, _ []string) error {
	sourcePath, err := filepath.Abs(appsPath)
	if err != nil {
		return fmt.Errorf("resolve path %q: %w", appsPath, err)
	}

	bundleDir, err := os.MkdirTemp("", "toolbox-apps-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(bundleDir)

	bundle, err := appbundle.WriteBundle(bundleDir, sourcePath, appsRepository, appsTag)
	if err != nil {
		return fmt.Errorf("write bundle: %w", err)
	}

	log.Infof("bundled %d app manifest file(s) into %d app artifact(s)", bundle.Count, len(bundle.Apps))

	tunnel, err := startKubectlPortForward(cmd.Context(), registryNamespace, registryService, registryPort)
	if err != nil {
		return fmt.Errorf("forward registry: %w", err)
	}
	defer tunnel.Close()

	for _, app := range bundle.Apps {
		if err := pushArtifactToRegistry(cmd.Context(), tunnel.addr, app.Dir, artifactRef{
			Repository:    app.Repository,
			Tag:           appsTag,
			Source:        app.Name,
			Revision:      appsTag,
			Kustomization: app.Name,
		}); err != nil {
			return fmt.Errorf("push %s: %w", app.Name, err)
		}
	}

	root := artifactRef{
		Repository:    appsRepository,
		Tag:           appsTag,
		Source:        appsRepository,
		Revision:      appsTag,
		Kustomization: appsRepository,
	}
	if err := pushArtifactToRegistry(cmd.Context(), tunnel.addr, bundle.RootDir, root); err != nil {
		return err
	}

	if err := reconcileArtifact(cmd.Context(), root); err != nil {
		return err
	}

	for _, app := range bundle.Apps {
		if err := reconcileArtifact(cmd.Context(), artifactRef{
			Source:        app.Name,
			Kustomization: app.Name,
		}); err != nil {
			return fmt.Errorf("reconcile %s: %w", app.Name, err)
		}
	}

	return nil
}

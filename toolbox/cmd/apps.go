package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	appbundle "github.com/khuedoan/cloudlab/toolbox/internal/apps"
)

const appsRepository = "apps"

var (
	appsEnv  string
	appsPath string
)

func init() {
	appsCmd.Flags().StringVar(&appsEnv, "env", "", "Environment to generate (for example: staging)")
	appsCmd.Flags().StringVar(&appsPath, "path", "apps", "Path to the app values tree")
	_ = appsCmd.MarkFlagRequired("env")
}

var appsCmd = &cobra.Command{
	Use:   "apps",
	Args:  cobra.NoArgs,
	Short: "Generate and push the apps manifest bundle",
	PreRunE: func(_ *cobra.Command, _ []string) error {
		if err := validateClusterFlags(); err != nil {
			return err
		}
		if _, err := exec.LookPath("flux"); err != nil {
			return fmt.Errorf("find flux CLI: %w", err)
		}
		return nil
	},
	RunE: runApps,
}

func runApps(cmd *cobra.Command, _ []string) error {
	sourcePath, err := filepath.Abs(appsPath)
	if err != nil {
		return fmt.Errorf("resolve path %q: %w", appsPath, err)
	}

	releases, err := appbundle.Discover(sourcePath, appsEnv)
	if err != nil {
		return fmt.Errorf("discover apps: %w", err)
	}

	bundleDir, err := os.MkdirTemp("", "toolbox-apps-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(bundleDir)

	if err := appbundle.WriteBundle(bundleDir, releases); err != nil {
		return fmt.Errorf("write bundle: %w", err)
	}

	log.Infof("generated %d app HelmRelease(s) for %s", len(releases), appsEnv)

	return pushArtifact(cmd.Context(), bundleDir, artifactRef{
		Repository:    appsRepository,
		Tag:           appsEnv,
		Source:        appsRepository,
		Revision:      appsEnv,
		Kustomization: appsRepository,
	})
}

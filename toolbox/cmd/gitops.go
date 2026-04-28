package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

const (
	gitopsRepository = "platform"
	gitopsTag        = "latest"
	gitopsSource     = "platform"
	gitopsBundle     = "platform"
)

var (
	gitopsPath string
)

func init() {
	gitopsCmd.Flags().StringVar(&gitopsPath, "path", "", "Path to the manifest bundle to publish")
	_ = gitopsCmd.MarkFlagRequired("path")
}

var gitopsCmd = &cobra.Command{
	Use:   "gitops",
	Short: "Proxy the in-cluster registry and push the GitOps manifests artifact",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := validateClusterFlags(); err != nil {
			return err
		}
		if _, err := exec.LookPath("flux"); err != nil {
			return fmt.Errorf("find flux CLI: %w", err)
		}
		return nil
	},
	RunE: runGitopsPush,
}

func runGitopsPush(cmd *cobra.Command, args []string) error {
	manifestPath, err := filepath.Abs(gitopsPath)
	if err != nil {
		return fmt.Errorf("resolve path %q: %w", gitopsPath, err)
	}

	return pushArtifact(cmd.Context(), manifestPath, artifactRef{
		Repository:    gitopsRepository,
		Tag:           gitopsTag,
		Source:        gitopsSource,
		Revision:      gitopsTag,
		Kustomization: gitopsBundle,
	})
}

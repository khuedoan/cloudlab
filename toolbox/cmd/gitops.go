package cmd

import (
	"fmt"
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
	Short: "Push the GitOps manifests artifact",
	PreRunE: func(_ *cobra.Command, _ []string) error {
		return requireExecutables("flux", "kubectl")
	},
	RunE: runGitopsPush,
}

func runGitopsPush(cmd *cobra.Command, _ []string) error {
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

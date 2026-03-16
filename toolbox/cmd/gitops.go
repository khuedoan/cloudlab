package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/khuedoan/cloudlab/toolbox/internal/cluster"
)

const (
	registryNamespace = "registry"
	registryService   = "svc/registry"
	registryPort      = 5000
	gitopsRepository  = "platform"
	gitopsTag         = "latest"
	fluxNamespace     = "flux-system"
	fluxSource        = "platform"
	fluxKustomization = "platform"
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

	artifactURL := fmt.Sprintf("oci://%s/%s:%s", tunnel.LocalAddr, gitopsRepository, gitopsTag)
	args = []string{
		"push",
		"artifact",
		artifactURL,
		"--path", manifestPath,
		// TODO should be actual source and revision
		"--source", "platform",
		"--revision", "latest",
		"--insecure-registry",
	}

	log.Infof("pushing %s from %s", artifactURL, manifestPath)

	output, err := fluxOutput(cmd.Context(), args...)
	if err != nil {
		return fmt.Errorf("push artifact: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
		log.Info(trimmed)
	}

	requestedAt := time.Now().UTC().Format(time.RFC3339Nano)
	log.Infof("triggering Flux sync for %s/%s", fluxNamespace, fluxKustomization)

	output, err = conn.RunCommandContext(
		cmd.Context(),
		fmt.Sprintf(
			"kubectl annotate --overwrite -n %s ocirepository.source.toolkit.fluxcd.io/%s reconcile.fluxcd.io/requestedAt=%q && kubectl annotate --overwrite -n %s kustomization.kustomize.toolkit.fluxcd.io/%s reconcile.fluxcd.io/requestedAt=%q",
			fluxNamespace, fluxSource, requestedAt,
			fluxNamespace, fluxKustomization, requestedAt,
		),
	)
	if err != nil {
		return fmt.Errorf("trigger flux sync: %w", err)
	}

	if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
		log.Info(trimmed)
	}

	return nil
}

func fluxOutput(ctx context.Context, args ...string) ([]byte, error) {
	fluxCmd := exec.CommandContext(ctx, "flux", args...)
	return fluxCmd.CombinedOutput()
}

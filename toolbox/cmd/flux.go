package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"github.com/khuedoan/cloudlab/toolbox/internal/cluster"
)

const (
	registryNamespace = "registry"
	registryService   = "svc/registry"
	registryPort      = 5000
	fluxNamespace     = "flux-system"
)

type artifactRef struct {
	Repository    string
	Tag           string
	Source        string
	Revision      string
	Kustomization string
}

func pushArtifact(ctx context.Context, manifestPath string, artifact artifactRef) error {
	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
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

	artifactURL := fmt.Sprintf("oci://%s/%s:%s", tunnel.LocalAddr, artifact.Repository, artifact.Tag)
	args := []string{
		"push",
		"artifact",
		artifactURL,
		"--path", manifestPath,
		"--source", artifact.Source,
		"--revision", artifact.Revision,
		"--insecure-registry",
	}

	log.Infof("pushing %s from %s", artifactURL, manifestPath)

	output, err := fluxOutput(ctx, args...)
	if err != nil {
		return fmt.Errorf("push artifact: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
		log.Info(trimmed)
	}

	requestedAt := time.Now().UTC().Format(time.RFC3339Nano)
	log.Infof("triggering Flux sync for %s/%s", fluxNamespace, artifact.Kustomization)

	output, err = conn.RunCommandContext(
		ctx,
		fmt.Sprintf(
			"kubectl annotate --overwrite -n %s ocirepository.source.toolkit.fluxcd.io/%s reconcile.fluxcd.io/requestedAt=%q && kubectl annotate --overwrite -n %s kustomization.kustomize.toolkit.fluxcd.io/%s reconcile.fluxcd.io/requestedAt=%q",
			fluxNamespace, artifact.Source,
			requestedAt,
			fluxNamespace, artifact.Kustomization,
			requestedAt,
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

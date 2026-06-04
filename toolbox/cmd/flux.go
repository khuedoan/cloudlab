package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/log"
)

const (
	registryNamespace = "registry"
	registryService   = "svc/registry"
	registryPort      = 80
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
	tunnel, err := startKubectlPortForward(ctx, registryNamespace, registryService, registryPort)
	if err != nil {
		return fmt.Errorf("forward registry: %w", err)
	}
	defer tunnel.Close()

	if err := pushArtifactToRegistry(ctx, tunnel.addr, manifestPath, artifact); err != nil {
		return err
	}

	return reconcileArtifact(ctx, artifact)
}

func pushArtifactToRegistry(ctx context.Context, registryAddr, manifestPath string, artifact artifactRef) error {
	artifactURL := fmt.Sprintf("oci://%s/%s:%s", registryAddr, artifact.Repository, artifact.Tag)
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

	return nil
}

func reconcileArtifact(ctx context.Context, artifact artifactRef) error {
	log.Infof("triggering Flux sync for %s/%s", fluxNamespace, artifact.Kustomization)

	output, err := fluxOutput(ctx, "reconcile", "source", "oci", artifact.Source, "--namespace", fluxNamespace)
	if err != nil {
		return fmt.Errorf("trigger OCIRepository sync: %w", err)
	}
	logCommandOutput(output)

	output, err = fluxOutput(ctx, "reconcile", "kustomization", artifact.Kustomization, "--namespace", fluxNamespace)
	if err != nil {
		return fmt.Errorf("trigger Kustomization sync: %w", err)
	}
	logCommandOutput(output)

	return nil
}

func fluxOutput(ctx context.Context, args ...string) ([]byte, error) {
	fluxCmd := exec.CommandContext(ctx, "flux", args...)
	return fluxCmd.CombinedOutput()
}

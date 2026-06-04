package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/khuedoan/cloudlab/toolbox/internal/backup"
)

type fluxResource struct {
	kind string
	name string
}

func suspendFlux(ctx context.Context, volumes []backup.Volume) ([]fluxResource, error) {
	var suspended []fluxResource
	for _, resource := range targetFluxResources(targetNamespaces(volumes)) {
		active, err := fluxResourceActive(ctx, resource)
		if err != nil {
			return nil, err
		}
		if !active {
			continue
		}
		if err := patchFluxResource(ctx, resource, true); err != nil {
			return nil, err
		}
		suspended = append(suspended, resource)
	}
	return suspended, nil
}

func resumeFlux(ctx context.Context, resources []fluxResource) error {
	for _, resource := range resources {
		if err := patchFluxResource(ctx, resource, false); err != nil {
			return err
		}
	}
	return nil
}

func targetFluxResources(namespaces []string) []fluxResource {
	resources := make([]fluxResource, 0, len(namespaces)*2)
	for _, namespace := range namespaces {
		resources = append(
			resources,
			fluxResource{kind: "helmrelease", name: namespace},
			fluxResource{kind: "kustomization", name: namespace},
		)
	}
	return resources
}

func fluxResourceActive(ctx context.Context, resource fluxResource) (bool, error) {
	output, err := runKubectl(ctx, "-n", fluxNamespace, "get", resource.kind, resource.name, "-o", "jsonpath={.spec.suspend}")
	if err != nil {
		if strings.Contains(string(output), "NotFound") {
			return false, nil
		}
		return false, fmt.Errorf("get %s %s: %w", resource.kind, resource.name, err)
	}
	return strings.TrimSpace(string(output)) != "true", nil
}

func patchFluxResource(ctx context.Context, resource fluxResource, suspend bool) error {
	patch := fmt.Sprintf(`{"spec":{"suspend":%t}}`, suspend)
	output, err := runKubectl(ctx, "-n", fluxNamespace, "patch", resource.kind, resource.name, "--type=merge", "-p", patch)
	if err != nil {
		return fmt.Errorf("patch %s %s: %w", resource.kind, resource.name, err)
	}
	logCommandOutput(output)
	return nil
}

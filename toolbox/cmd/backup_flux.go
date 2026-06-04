package cmd

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/khuedoan/cloudlab/toolbox/internal/backup"
)

func suspendFlux(ctx context.Context, volumes []backup.Volume) ([]string, error) {
	releases, err := targetHelmReleases(ctx, volumes)
	if err != nil {
		return nil, err
	}
	for _, name := range releases {
		if err := patchHelmRelease(ctx, name, true); err != nil {
			return nil, err
		}
	}
	return releases, nil
}

func resumeFlux(ctx context.Context, releases []string) error {
	for _, name := range releases {
		if err := patchHelmRelease(ctx, name, false); err != nil {
			return err
		}
	}
	return nil
}

func targetHelmReleases(ctx context.Context, volumes []backup.Volume) ([]string, error) {
	output, err := runKubectl(ctx, "-n", fluxNamespace, "get", "helmrelease", "-o", `jsonpath={range .items[*]}{.metadata.name}{"\t"}{.spec.targetNamespace}{"\t"}{.metadata.namespace}{"\t"}{.spec.suspend}{"\n"}{end}`)
	if err != nil {
		return nil, fmt.Errorf("list HelmReleases: %w", err)
	}

	namespaces := targetNamespaces(volumes)

	names := make([]string, 0)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 4 {
			continue
		}
		targetNamespace := fields[1]
		if targetNamespace == "" {
			targetNamespace = fields[2]
		}
		if slices.Contains(namespaces, targetNamespace) && fields[3] != "true" {
			names = append(names, fields[0])
		}
	}
	slices.Sort(names)
	return names, nil
}

func patchHelmRelease(ctx context.Context, name string, suspend bool) error {
	patch := fmt.Sprintf(`{"spec":{"suspend":%t}}`, suspend)
	output, err := runKubectl(ctx, "-n", fluxNamespace, "patch", "helmrelease", name, "--type=merge", "-p", patch)
	if err != nil {
		return fmt.Errorf("patch HelmRelease %s: %w", name, err)
	}
	logCommandOutput(output)
	return nil
}

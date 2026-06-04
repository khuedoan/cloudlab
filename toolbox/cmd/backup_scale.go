package cmd

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/khuedoan/cloudlab/toolbox/internal/backup"
)

func scaleRestoreNamespaces(ctx context.Context, volumes []backup.Volume, replicas int) error {
	for _, namespace := range targetNamespaces(volumes) {
		if err := scaleNamespaceWorkloads(ctx, namespace, replicas); err != nil {
			return err
		}
	}
	return nil
}

func scaleNamespaceWorkloads(ctx context.Context, namespace string, replicas int) error {
	output, err := runKubectl(ctx, "-n", namespace, "scale", "deployment,statefulset", "--all", fmt.Sprintf("--replicas=%d", replicas))
	if err != nil {
		return fmt.Errorf("scale workloads in %s to %d replica(s): %w", namespace, replicas, err)
	}
	logCommandOutput(output)
	return nil
}

func targetNamespaces(volumes []backup.Volume) []string {
	namespaces := map[string]bool{}
	for _, volume := range volumes {
		namespaces[volume.Namespace] = true
	}
	return slices.Sorted(maps.Keys(namespaces))
}

func waitForPodsDetached(ctx context.Context, volumes []backup.Volume) error {
	for _, namespace := range targetNamespaces(volumes) {
		output, err := runKubectl(ctx, "-n", namespace, "wait", "--for=delete", "pod", "--all", "--timeout="+podDetachTimeout.String())
		if err != nil {
			return fmt.Errorf("wait for pods in %s to stop: %w", namespace, err)
		}
		logCommandOutput(output)
	}
	return nil
}

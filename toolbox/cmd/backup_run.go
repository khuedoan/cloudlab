package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"

	"github.com/khuedoan/cloudlab/toolbox/internal/backup"
)

func prepareBackupRun(ctx context.Context, requireSourcePVC bool) ([]backup.Volume, error) {
	config, err := backup.LoadConfig(backupSettingsFile)
	if err != nil {
		return nil, fmt.Errorf("load settings file: %w", err)
	}

	volumes, err := backup.ParseAndValidate(config)
	if err != nil {
		return nil, fmt.Errorf("validate backup inventory: %w", err)
	}
	volumes, err = backup.FilterVolumes(volumes, backupVolumeSelectors)
	if err != nil {
		return nil, fmt.Errorf("select backup volumes: %w", err)
	}

	if requireSourcePVC {
		if err := ensurePVCs(ctx, volumes, "source"); err != nil {
			return nil, err
		}
	}

	return volumes, nil
}

func applyBackupObjects(ctx context.Context, objects []backup.Object) error {
	manifest, err := backup.RenderYAML(objects)
	if err != nil {
		return fmt.Errorf("render backup manifests: %w", err)
	}

	log.Infof("generated %d Kubernetes object(s) for %s", len(objects), backupEnv)

	if err := applyManifest(ctx, manifest, "apply", "--server-side", "--dry-run=server", "-f", "-"); err != nil {
		return fmt.Errorf("server-side dry-run: %w", err)
	}
	if err := applyManifest(ctx, manifest, "apply", "--server-side", "-f", "-"); err != nil {
		return fmt.Errorf("apply manifests: %w", err)
	}

	return nil
}

func ensurePVCs(ctx context.Context, volumes []backup.Volume, role string) error {
	for _, volume := range volumes {
		if _, err := runKubectl(ctx, "-n", volume.Namespace, "get", "pvc", volume.PVC); err != nil {
			return fmt.Errorf("%s PVC %s/%s is not ready: %w", role, volume.Namespace, volume.PVC, err)
		}
	}
	return nil
}

func applyManifest(ctx context.Context, manifest []byte, args ...string) error {
	output, err := runKubectlInput(ctx, manifest, args...)
	if err != nil {
		return fmt.Errorf("kubectl %s: %w (output: %s)", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}

	logCommandOutput(output)
	return nil
}

func logCommandOutput(output []byte) {
	if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
		log.Info(trimmed)
	}
}

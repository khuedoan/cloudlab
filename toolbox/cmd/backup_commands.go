package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/khuedoan/cloudlab/toolbox/internal/backup"
)

func runBackupSetup(cmd *cobra.Command, _ []string) error {
	volumes, err := prepareBackupRun(cmd.Context(), true)
	if err != nil {
		return err
	}

	objects := backup.BuildSetupObjects(volumes)
	if err := applyBackupObjects(cmd.Context(), objects); err != nil {
		return err
	}

	log.Info("backup resources applied successfully")
	return nil
}

func runBackupRestore(cmd *cobra.Command, _ []string) error {
	restoreTrigger := "restore-" + time.Now().UTC().Format("20060102T150405.000000000Z")
	log.Infof("using restore trigger %s", restoreTrigger)

	volumes, err := prepareBackupRun(cmd.Context(), false)
	if err != nil {
		return err
	}

	if err := ensurePVCs(cmd.Context(), volumes, "destination"); err != nil {
		return err
	}

	objects := backup.BuildRestoreObjects(volumes, restoreTrigger)

	suspendedHelmReleases, err := suspendFlux(cmd.Context(), volumes)
	if err != nil {
		return fmt.Errorf("suspend Flux: %w", err)
	}
	log.Info("Flux HelmRelease reconciliation suspended")

	if err := scaleRestoreNamespaces(cmd.Context(), volumes, 0); err != nil {
		return restorePausedError(err)
	}
	if err := waitForPodsDetached(cmd.Context(), volumes); err != nil {
		return restorePausedError(err)
	}
	if err := applyBackupObjects(cmd.Context(), objects); err != nil {
		return restorePausedError(err)
	}
	if err := waitForRestores(cmd.Context(), volumes, restoreTrigger); err != nil {
		return restorePausedError(err)
	}
	if err := scaleRestoreNamespaces(cmd.Context(), volumes, 1); err != nil {
		return restorePausedError(err)
	}
	if err := resumeFlux(cmd.Context(), suspendedHelmReleases); err != nil {
		return fmt.Errorf("resume Flux after successful restore: %w", err)
	}

	log.Info("restore completed successfully")
	return nil
}

func waitForRestores(ctx context.Context, volumes []backup.Volume, restoreTrigger string) error {
	for _, volume := range volumes {
		name := backup.DestinationName(volume)
		for _, wait := range []string{
			"--for=jsonpath={.status.lastManualSync}=" + restoreTrigger,
			"--for=jsonpath={.status.conditions[?(@.type==\"Synchronizing\")].reason}=WaitingForManual",
		} {
			output, err := runKubectl(ctx, "-n", volume.Namespace, "wait", wait, "replicationdestination/"+name, "--timeout="+restoreTimeout.String())
			if err != nil {
				return fmt.Errorf("wait for restore %s/%s: %w", volume.Namespace, name, err)
			}
			logCommandOutput(output)
		}
		log.Infof("restore completed for %s/%s", volume.Namespace, name)
	}
	return nil
}

func restorePausedError(err error) error {
	return fmt.Errorf("%w; Flux remains suspended and restored workloads remain scaled down for inspection; restore Flux suspension state manually after recovery", err)
}

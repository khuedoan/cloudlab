package vendors

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

func LoadVendors(configPath string) ([]VendorEntry, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("load settings file: %w", err)
	}

	entries, err := ParseAndValidate(config)
	if err != nil {
		return nil, fmt.Errorf("validate settings: %w", err)
	}

	return entries, nil
}

func Sync(ctx context.Context, workdir, registryAddr string, entries []VendorEntry) error {
	for _, item := range entries {
		switch item.Kind {
		case "chart":
			if err := syncChart(ctx, workdir, registryAddr, item); err != nil {
				return err
			}
		case "image":
			if err := syncImage(ctx, registryAddr, item); err != nil {
				return err
			}
		}
	}
	return nil
}

func syncChart(ctx context.Context, workdir, registryAddr string, chart VendorEntry) error {
	chartDir := filepath.Join(workdir, chart.Name)
	if err := os.MkdirAll(chartDir, 0o755); err != nil {
		return fmt.Errorf("create chart temp dir: %w", err)
	}

	pullRef := chart.Chart
	if chart.Ref != "" {
		pullRef = chart.Ref
	}
	for _, version := range chart.Versions {
		log.Infof("vendoring chart %s@%s", chart.Name, version)

		pullArgs := []string{"pull", pullRef, "--version", version, "--destination", chartDir}
		if chart.RepoURL != "" {
			pullArgs = append(pullArgs, "--repo", chart.RepoURL)
		}
		if err := runCommand(ctx, "helm", pullArgs...); err != nil {
			return fmt.Errorf("pull chart %s@%s: %w", chart.Name, version, err)
		}

		archivePath := filepath.Join(chartDir, filepath.Base(pullRef)+"-"+version+".tgz")
		pushTarget := fmt.Sprintf("oci://%s/%s", registryAddr, chart.Name)
		if err := runCommand(ctx, "helm", "push", archivePath, pushTarget, "--plain-http"); err != nil {
			return fmt.Errorf("push chart %s@%s: %w", chart.Name, version, err)
		}
	}

	return nil
}

func syncImage(ctx context.Context, registryAddr string, image VendorEntry) error {
	for _, version := range image.Versions {
		log.Infof("vendoring image %s:%s", image.Name, version)

		source := image.Source
		target := image.Name
		if strings.HasPrefix(version, "@") {
			source += version
			target += version
		} else {
			source += ":" + version
			target += ":" + version
		}
		destination := fmt.Sprintf("%s/%s", registryAddr, target)
		copyArgs := []string{"cp", source, destination, "--to-plain-http"}

		if err := runCommand(ctx, "oras", copyArgs...); err != nil {
			return fmt.Errorf("copy image %s@%s: %w", image.Name, version, err)
		}
	}

	return nil
}

func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w", name, args, err)
	}

	return nil
}

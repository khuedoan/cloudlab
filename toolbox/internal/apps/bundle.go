package apps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	chartName            = "app-template"
	chartVersion         = "4.6.0"
	fluxNamespace        = "flux-system"
	helmRepositoryKind   = "HelmRepository"
	helmRepositoryName   = "app-template"
	helmReleaseKind      = "HelmRelease"
	helmReleaseVersion   = "helm.toolkit.fluxcd.io/v2"
	kustomizationKind    = "Kustomization"
	kustomizationFile    = "kustomization.yaml"
	kustomizationVersion = "kustomize.config.k8s.io/v1beta1"
	reconcileInterval    = "3m"
)

type Release struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Metadata   Metadata    `yaml:"metadata"`
	Spec       ReleaseSpec `yaml:"spec"`
}

type Metadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace,omitempty"`
}

type ReleaseSpec struct {
	Interval        string         `yaml:"interval"`
	ReleaseName     string         `yaml:"releaseName"`
	TargetNamespace string         `yaml:"targetNamespace"`
	Install         InstallSpec    `yaml:"install"`
	Chart           Chart          `yaml:"chart"`
	Values          map[string]any `yaml:"values,omitempty"`
}

type InstallSpec struct {
	CreateNamespace bool `yaml:"createNamespace"`
}

type Chart struct {
	Spec ChartSpec `yaml:"spec"`
}

type ChartSpec struct {
	Chart     string    `yaml:"chart"`
	Version   string    `yaml:"version"`
	SourceRef SourceRef `yaml:"sourceRef"`
}

type SourceRef struct {
	Kind string `yaml:"kind"`
	Name string `yaml:"name"`
}

type kustomization struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Resources  []string `yaml:"resources"`
}

func Discover(rootDir, env string) ([]Release, error) {
	pattern := filepath.Join(rootDir, "*", "*", env+".yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob app values: %w", err)
	}

	releases := make([]Release, 0, len(files))
	for _, path := range files {
		release, err := loadRelease(rootDir, path)
		if err != nil {
			return nil, err
		}
		releases = append(releases, release)
	}

	return releases, nil
}

func WriteBundle(outputDir string, releases []Release) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create bundle dir: %w", err)
	}

	resources := make([]string, 0, len(releases))
	for _, release := range releases {
		filename := release.Metadata.Name + ".yaml"
		path := filepath.Join(outputDir, filename)

		data, err := yaml.Marshal(release)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", release.Metadata.Name, err)
		}

		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}

		resources = append(resources, filename)
	}

	data, err := yaml.Marshal(kustomization{
		APIVersion: kustomizationVersion,
		Kind:       kustomizationKind,
		Resources:  resources,
	})
	if err != nil {
		return fmt.Errorf("marshal kustomization: %w", err)
	}

	path := filepath.Join(outputDir, kustomizationFile)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

func loadRelease(rootDir, path string) (Release, error) {
	relPath, err := filepath.Rel(rootDir, path)
	if err != nil {
		return Release{}, fmt.Errorf("resolve relative path for %s: %w", path, err)
	}

	parts := splitPath(relPath)
	if len(parts) != 3 {
		return Release{}, fmt.Errorf("expected apps/$NAMESPACE/$APP/$ENV.yaml, got %s", relPath)
	}

	valuesData, err := os.ReadFile(path)
	if err != nil {
		return Release{}, fmt.Errorf("read %s: %w", path, err)
	}

	values := map[string]any{}
	if len(valuesData) > 0 {
		if err := yaml.Unmarshal(valuesData, &values); err != nil {
			return Release{}, fmt.Errorf("decode %s: %w", path, err)
		}
	}

	namespace := parts[0]
	app := parts[1]

	return Release{
		APIVersion: helmReleaseVersion,
		Kind:       helmReleaseKind,
		Metadata: Metadata{
			Name:      namespace + "-" + app,
			Namespace: fluxNamespace,
		},
		Spec: ReleaseSpec{
			Interval:        reconcileInterval,
			ReleaseName:     app,
			TargetNamespace: namespace,
			Install: InstallSpec{
				CreateNamespace: true,
			},
			Chart: Chart{
				Spec: ChartSpec{
					Chart:   chartName,
					Version: chartVersion,
					SourceRef: SourceRef{
						Kind: helmRepositoryKind,
						Name: helmRepositoryName,
					},
				},
			},
			Values: values,
		},
	}, nil
}

func splitPath(path string) []string {
	return strings.Split(filepath.ToSlash(path), "/")
}

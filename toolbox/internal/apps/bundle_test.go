package apps

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDiscoverFiltersByEnvironment(t *testing.T) {
	rootDir := t.TempDir()

	writeFile(t, filepath.Join(rootDir, "khuedoan", "blog", "staging.yaml"), "controllers:\n  main:\n    replicas: 2\n")
	writeFile(t, filepath.Join(rootDir, "finance", "actualbudget", "staging.yaml"), "service:\n  main:\n    controller: main\n")
	writeFile(t, filepath.Join(rootDir, "test", "example", "production.yaml"), "ignored: true\n")

	releases, err := Discover(rootDir, "staging")
	if err != nil {
		t.Fatalf("discover staging apps: %v", err)
	}

	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}

	gotNames := []string{releases[0].Metadata.Name, releases[1].Metadata.Name}
	wantNames := []string{"finance-actualbudget", "khuedoan-blog"}
	if !slices.Equal(gotNames, wantNames) {
		t.Fatalf("expected release names %v, got %v", wantNames, gotNames)
	}

	if releases[0].Spec.TargetNamespace != "finance" {
		t.Fatalf("expected finance namespace, got %q", releases[0].Spec.TargetNamespace)
	}

	controllers, ok := releases[1].Spec.Values["controllers"].(map[string]any)
	if !ok {
		t.Fatalf("expected controllers map, got %T", releases[1].Spec.Values["controllers"])
	}

	main, ok := controllers["main"].(map[string]any)
	if !ok {
		t.Fatalf("expected main controller map, got %T", controllers["main"])
	}

	if main["replicas"] != 2 {
		t.Fatalf("expected replicas 2, got %#v", main["replicas"])
	}
}

func TestWriteBundleCreatesKustomizationAndResources(t *testing.T) {
	outputDir := t.TempDir()
	releases := []Release{
		{
			APIVersion: helmReleaseVersion,
			Kind:       helmReleaseKind,
			Metadata: Metadata{
				Name:      "khuedoan-blog",
				Namespace: fluxNamespace,
			},
			Spec: ReleaseSpec{
				Interval:        reconcileInterval,
				ReleaseName:     "blog",
				TargetNamespace: "khuedoan",
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
				Values: map[string]any{
					"service": map[string]any{
						"main": map[string]any{
							"controller": "main",
						},
					},
				},
			},
		},
	}

	if err := WriteBundle(outputDir, releases); err != nil {
		t.Fatalf("write bundle: %v", err)
	}

	kustomizationData, err := os.ReadFile(filepath.Join(outputDir, kustomizationFile))
	if err != nil {
		t.Fatalf("read kustomization: %v", err)
	}

	var manifest kustomization
	if err := yaml.Unmarshal(kustomizationData, &manifest); err != nil {
		t.Fatalf("decode kustomization: %v", err)
	}

	if !slices.Equal(manifest.Resources, []string{"khuedoan-blog.yaml"}) {
		t.Fatalf("expected resources [khuedoan-blog.yaml], got %v", manifest.Resources)
	}

	resourceData, err := os.ReadFile(filepath.Join(outputDir, "khuedoan-blog.yaml"))
	if err != nil {
		t.Fatalf("read release manifest: %v", err)
	}

	var release Release
	if err := yaml.Unmarshal(resourceData, &release); err != nil {
		t.Fatalf("decode release manifest: %v", err)
	}

	if release.Spec.ReleaseName != "blog" {
		t.Fatalf("expected release name blog, got %q", release.Spec.ReleaseName)
	}
}

func TestWriteBundleSupportsNoApps(t *testing.T) {
	outputDir := t.TempDir()

	if err := WriteBundle(outputDir, nil); err != nil {
		t.Fatalf("write empty bundle: %v", err)
	}

	kustomizationData, err := os.ReadFile(filepath.Join(outputDir, kustomizationFile))
	if err != nil {
		t.Fatalf("read kustomization: %v", err)
	}

	var manifest kustomization
	if err := yaml.Unmarshal(kustomizationData, &manifest); err != nil {
		t.Fatalf("decode kustomization: %v", err)
	}

	if len(manifest.Resources) != 0 {
		t.Fatalf("expected no resources, got %v", manifest.Resources)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create dir for %s: %v", path, err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

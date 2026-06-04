package apps

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	k8syaml "sigs.k8s.io/yaml"
)

const (
	appsRootDir       = "apps"
	fluxNamespace     = "flux-system"
	namespaceKind     = "Namespace"
	rootDir           = "root"
	kustomizationKind = "Kustomization"
	ociRepositoryKind = "OCIRepository"
	reconcileInterval = "1m"
	sourceInterval    = "30s"
	clusterRegistry   = "registry.registry.svc.cluster.local"
)

type Bundle struct {
	Apps    []AppBundle
	Count   int
	RootDir string
}

type AppBundle struct {
	Dir        string
	Name       string
	Repository string
}

func WriteBundle(outputDir, sourceDir, repository, tag string) (Bundle, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return Bundle{}, fmt.Errorf("create bundle dir: %w", err)
	}

	count := 0
	apps := map[string]AppBundle{}
	err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !isYAML(path) {
			return nil
		}
		if isKustomization(path) {
			return fmt.Errorf("%s is not supported; put plain Kubernetes YAML in apps instead", path)
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("resolve relative path for %s: %w", path, err)
		}
		appEnv, err := appEnvPath(relPath)
		if err != nil {
			return err
		}
		app := appBundle(outputDir, repository, appEnv)

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		manifest, err := validateManifest(path, data)
		if err != nil {
			return err
		}
		if err := validateFilename(path, manifest); err != nil {
			return err
		}
		if err := validateNamespace(path, manifest, app.Name); err != nil {
			return err
		}

		outputPath := filepath.Join(app.Dir, filepath.Base(relPath))
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return fmt.Errorf("create dir for %s: %w", outputPath, err)
		}
		if err := os.WriteFile(outputPath, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", outputPath, err)
		}

		count++
		apps[appEnv] = app
		return nil
	})
	if err != nil {
		return Bundle{}, fmt.Errorf("copy app manifests: %w", err)
	}

	if count == 0 {
		return Bundle{}, fmt.Errorf("no app manifests found in %s", sourceDir)
	}

	bundle := Bundle{
		Apps:    sortedApps(apps),
		Count:   count,
		RootDir: filepath.Join(outputDir, rootDir),
	}
	if err := writeRootBundle(bundle.RootDir, bundle.Apps, tag); err != nil {
		return Bundle{}, err
	}

	return bundle, nil
}

func validateManifest(path string, data []byte) (*unstructured.Unstructured, error) {
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	var manifest *unstructured.Unstructured

	for {
		object := map[string]any{}
		err := decoder.Decode(&object)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		if len(object) == 0 {
			continue
		}
		if manifest != nil {
			return nil, fmt.Errorf("%s: expected one Kubernetes manifest per file", path)
		}

		manifest = &unstructured.Unstructured{Object: object}
		switch {
		case manifest.GetAPIVersion() == "":
			return nil, fmt.Errorf("%s: manifest is missing apiVersion", path)
		case manifest.GetKind() == "":
			return nil, fmt.Errorf("%s: manifest is missing kind", path)
		case manifest.GetName() == "":
			return nil, fmt.Errorf("%s: %s is missing metadata.name", path, manifest.GetKind())
		}
	}

	if manifest == nil {
		return nil, fmt.Errorf("%s: no Kubernetes manifests found", path)
	}

	return manifest, nil
}

func validateFilename(path string, manifest *unstructured.Unstructured) error {
	want := strings.ToLower(manifest.GetKind()) + "-" + manifest.GetName() + ".yaml"
	if filepath.Base(path) != want {
		return fmt.Errorf("%s: expected filename %s", path, want)
	}

	return nil
}

func validateNamespace(path string, manifest *unstructured.Unstructured, namespace string) error {
	if manifest.GetKind() == namespaceKind {
		if manifest.GetName() != namespace {
			return fmt.Errorf("%s: Namespace name must be %q", path, namespace)
		}
		return nil
	}

	if manifest.GetNamespace() != namespace {
		return fmt.Errorf("%s: %s/%s metadata.namespace must be %q", path, manifest.GetKind(), manifest.GetName(), namespace)
	}

	return nil
}

func appEnvPath(path string) (string, error) {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) != 4 {
		return "", fmt.Errorf("%s: expected apps/$TENANT/$PROJECT/$APP_ENV/$RESOURCE.yaml", path)
	}

	return strings.Join(parts[:3], "/"), nil
}

func appBundle(outputDir, repository, appEnv string) AppBundle {
	name := strings.ReplaceAll(appEnv, "/", "-")
	return AppBundle{
		Dir:        filepath.Join(outputDir, appsRootDir, appEnv),
		Name:       name,
		Repository: repository + "/" + appEnv,
	}
}

func sortedApps(apps map[string]AppBundle) []AppBundle {
	paths := slices.Sorted(maps.Keys(apps))
	bundles := make([]AppBundle, 0, len(paths))
	for _, path := range paths {
		bundles = append(bundles, apps[path])
	}
	return bundles
}

func writeRootBundle(outputDir string, apps []AppBundle, tag string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create root bundle dir: %w", err)
	}

	for _, app := range apps {
		manifests := []map[string]any{
			{
				"apiVersion": "source.toolkit.fluxcd.io/v1",
				"kind":       ociRepositoryKind,
				"metadata": map[string]any{
					"name":      app.Name,
					"namespace": fluxNamespace,
				},
				"spec": map[string]any{
					"interval": sourceInterval,
					"url":      "oci://" + clusterRegistry + "/" + app.Repository,
					"insecure": true,
					"ref": map[string]string{
						"tag": tag,
					},
				},
			},
			{
				"apiVersion": "kustomize.toolkit.fluxcd.io/v1",
				"kind":       kustomizationKind,
				"metadata": map[string]any{
					"name":      app.Name,
					"namespace": fluxNamespace,
				},
				"spec": map[string]any{
					"interval": reconcileInterval,
					"dependsOn": []map[string]string{
						{"name": "platform"},
					},
					"path":  ".",
					"prune": true,
					"sourceRef": map[string]string{
						"kind": "OCIRepository",
						"name": app.Name,
					},
				},
			},
		}

		for _, manifest := range manifests {
			kind, _ := manifest["kind"].(string)
			filename := strings.ToLower(kind) + "-" + app.Name + ".yaml"
			data, err := k8syaml.Marshal(manifest)
			if err != nil {
				return fmt.Errorf("marshal %s: %w", filename, err)
			}

			path := filepath.Join(outputDir, filename)
			if err := os.WriteFile(path, data, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}
		}
	}

	return nil
}

func isYAML(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

func isKustomization(path string) bool {
	switch strings.ToLower(filepath.Base(path)) {
	case "kustomization.yaml", "kustomization.yml":
		return true
	default:
		return false
	}
}

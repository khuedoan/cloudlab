package apps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteBundleCopiesPlainManifests(t *testing.T) {
	sourceDir := t.TempDir()
	outputDir := t.TempDir()
	manifest := `apiVersion: v1
kind: Namespace
metadata:
  name: khuedoan-blog-staging
`

	writeFile(t, filepath.Join(sourceDir, "khuedoan", "blog", "staging", "namespace-khuedoan-blog-staging.yaml"), manifest)
	writeFile(t, filepath.Join(sourceDir, "notes.md"), "# ignored\n")

	bundle, err := WriteBundle(outputDir, sourceDir, "apps", "latest")
	if err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	if bundle.Count != 1 || len(bundle.Apps) != 1 {
		t.Fatalf("expected 1 manifest and 1 app, got %d manifest(s) and %d app(s)", bundle.Count, len(bundle.Apps))
	}

	app := bundle.Apps[0]
	if app.Name != "khuedoan-blog-staging" || app.Repository != "apps/khuedoan/blog/staging" {
		t.Fatalf("unexpected app bundle: %#v", app)
	}

	copied, err := os.ReadFile(filepath.Join(app.Dir, "namespace-khuedoan-blog-staging.yaml"))
	if err != nil {
		t.Fatalf("read copied manifest: %v", err)
	}
	if string(copied) != manifest {
		t.Fatalf("copied manifest changed:\n%s", copied)
	}

	source := readFile(t, filepath.Join(bundle.RootDir, "ocirepository-khuedoan-blog-staging.yaml"))
	if !strings.Contains(source, "url: oci://registry.registry.svc.cluster.local/apps/khuedoan/blog/staging") {
		t.Fatalf("generated source has wrong URL:\n%s", source)
	}

	kustomization := readFile(t, filepath.Join(bundle.RootDir, "kustomization-khuedoan-blog-staging.yaml"))
	if !strings.Contains(kustomization, "path: .") || !strings.Contains(kustomization, "name: khuedoan-blog-staging") {
		t.Fatalf("generated kustomization has wrong source/path:\n%s", kustomization)
	}
}

func TestWriteBundleRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		content string
		wantErr string
	}{
		{
			name:    "non manifest YAML",
			path:    "khuedoan/blog/staging/namespace-khuedoan-blog-staging.yaml",
			content: "controllers: {}\n",
			wantErr: "missing apiVersion",
		},
		{
			name: "kustomization file",
			path: "khuedoan/blog/staging/kustomization.yaml",
			content: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: blog
`,
			wantErr: "not supported",
		},
		{
			name: "multiple manifests",
			path: "khuedoan/blog/staging/namespace-khuedoan-blog-staging.yaml",
			content: `apiVersion: v1
kind: Namespace
metadata:
  name: khuedoan-blog-staging
---
apiVersion: v1
kind: Service
metadata:
  name: blog
`,
			wantErr: "expected one Kubernetes manifest per file",
		},
		{
			name: "wrong filename",
			path: "khuedoan/blog/staging/blog.yaml",
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: blog
`,
			wantErr: "expected filename deployment-blog.yaml",
		},
		{
			name: "namespace object outside app namespace",
			path: "khuedoan/blog/staging/namespace-other.yaml",
			content: `apiVersion: v1
kind: Namespace
metadata:
  name: other
`,
			wantErr: `Namespace name must be "khuedoan-blog-staging"`,
		},
		{
			name: "namespaced object outside app namespace",
			path: "khuedoan/blog/staging/deployment-blog.yaml",
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: blog
  namespace: other
`,
			wantErr: `metadata.namespace must be "khuedoan-blog-staging"`,
		},
		{
			name: "namespaced object missing namespace",
			path: "khuedoan/blog/staging/deployment-blog.yaml",
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: blog
`,
			wantErr: `metadata.namespace must be "khuedoan-blog-staging"`,
		},
		{
			name: "unexpected path",
			path: "khuedoan/blog/deployment-blog.yaml",
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: blog
`,
			wantErr: "expected apps/$TENANT/$PROJECT/$APP_ENV/$RESOURCE.yaml",
		},
		{
			name:    "empty tree",
			wantErr: "no app manifests found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sourceDir := t.TempDir()
			if tc.path != "" {
				writeFile(t, filepath.Join(sourceDir, tc.path), tc.content)
			}

			_, err := WriteBundle(t.TempDir(), sourceDir, "apps", "latest")
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
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

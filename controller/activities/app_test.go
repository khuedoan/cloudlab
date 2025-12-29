package activities

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestUpdateAppVersion(t *testing.T) {
	tests := []struct {
		name           string
		yamlContent    string
		newImages      []Image
		expectedUpdate bool
		expectError    bool
	}{
		{
			name: "blog app production update",
			yamlContent: `defaultPodOptions:
  labels:
    "istio.io/dataplane-mode": "ambient"
controllers:
  main:
    replicas: 2
    strategy: RollingUpdate
    containers:
      main:
        image:
          repository: docker.io/khuedoan/blog
          tag: 6fbd90b77a81e0bcb330fddaa230feff744a7010
service:
  main:
    controller: main
    ports:
      http:
        port: 3000
        protocol: HTTP`,
			newImages: []Image{
				{Repository: "docker.io/khuedoan/blog", Tag: "abc123def456789"},
			},
			expectedUpdate: true,
		},
		{
			name: "actualbudget app version update",
			yamlContent: `defaultPodOptions:
  labels:
    "istio.io/dataplane-mode": "ambient"
controllers:
  main:
    containers:
      main:
        image:
          repository: docker.io/actualbudget/actual-server
          tag: 25.6.1-alpine
service:
  main:
    controller: main
    ports:
      http:
        port: 5006
        protocol: HTTP`,
			newImages: []Image{
				{Repository: "docker.io/actualbudget/actual-server", Tag: "25.7.0-alpine"},
			},
			expectedUpdate: true,
		},
		{
			name: "notes app with ghcr registry",
			yamlContent: `defaultPodOptions:
  labels:
    istio.io/dataplane-mode: ambient
controllers:
  main:
    type: statefulset
    containers:
      main:
        image:
          repository: ghcr.io/silverbulletmd/silverbullet
          tag: v2
        envFrom:
          - secret: silverbullet
service:
  main:
    controller: main
    ports:
      http:
        port: 3000
        protocol: HTTP`,
			newImages: []Image{
				{Repository: "ghcr.io/silverbulletmd/silverbullet", Tag: "v3"},
			},
			expectedUpdate: true,
		},
		{
			name: "example service with local registry",
			yamlContent: `defaultPodOptions:
  labels:
    istio.io/dataplane-mode: ambient
controllers:
  main:
    replicas: 2
    strategy: RollingUpdate
    containers:
      main:
        image:
          repository: registry.registry.svc.cluster.local/example-service
          tag: 828c31f942e8913ab2af53a2841c180586c5b7e1
service:
  main:
    controller: main
    ports:
      http:
        port: 8080
        protocol: HTTP`,
			newImages: []Image{
				{Repository: "registry.registry.svc.cluster.local/example-service", Tag: "abc123def456789012345678901234567890abcd"},
			},
			expectedUpdate: true,
		},
		{
			name: "no matching repository",
			yamlContent: `defaultPodOptions:
  labels:
    istio.io/dataplane-mode: ambient
controllers:
  main:
    containers:
      main:
        image:
          repository: docker.io/khuedoan/blog
          tag: 6fbd90b77a81e0bcb330fddaa230feff744a7010`,
			newImages: []Image{
				{Repository: "docker.io/different/app", Tag: "newversion"},
			},
			expectedUpdate: false,
		},
		{
			name: "multiple images same yaml - partial update",
			yamlContent: `defaultPodOptions:
  labels:
    istio.io/dataplane-mode: ambient
controllers:
  frontend:
    containers:
      main:
        image:
          repository: docker.io/khuedoan/blog
          tag: 6fbd90b77a81e0bcb330fddaa230feff744a7010
  backend:
    containers:
      main:
        image:
          repository: ghcr.io/silverbulletmd/silverbullet
          tag: v2`,
			newImages: []Image{
				{Repository: "docker.io/khuedoan/blog", Tag: "newcommithash123"},
			},
			expectedUpdate: true,
		},
		{
			name: "malformed yaml structure",
			yamlContent: `controllers:
  main:
    containers:
      main:
        image:
          repository: docker.io/test/app
        # missing tag field
service:
  main: invalid yaml structure`,
			newImages: []Image{
				{Repository: "docker.io/test/app", Tag: "v2.0.0"},
			},
			expectError: false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			tempDir, err := os.MkdirTemp("", "test-update-app-")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			namespace := "test-ns"
			app := "test-app"
			cluster := "test-cluster"

			// Create directory structure
			appDir := filepath.Join(tempDir, namespace, app)
			err = os.MkdirAll(appDir, 0755)
			require.NoError(t, err)

			// Write test YAML file
			yamlPath := filepath.Join(appDir, fmt.Sprintf("%s.yaml", cluster))
			err = os.WriteFile(yamlPath, []byte(tt.yamlContent), 0644)
			require.NoError(t, err)

			// Execute UpdateAppVersion
			ctx := context.Background()
			changed, err := UpdateAppVersion(ctx, tempDir, namespace, app, cluster, tt.newImages)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedUpdate, changed, "Expected change result doesn't match")

			// Read the updated file
			updatedContent, err := os.ReadFile(yamlPath)
			require.NoError(t, err)

			// Parse the updated YAML
			var updatedData map[string]interface{}
			err = yaml.Unmarshal(updatedContent, &updatedData)
			require.NoError(t, err)

			// Verify updates were applied correctly
			if tt.expectedUpdate {
				verifyImageUpdates(t, updatedData, tt.newImages)
			}
		})
	}
}

func TestUpdateAppVersion_FileErrors(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "test-update-app-errors-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("non-existent file", func(t *testing.T) {
		_, err := UpdateAppVersion(ctx, tempDir, "ns", "app", "cluster", []Image{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read file")
	})

	t.Run("invalid yaml", func(t *testing.T) {
		namespace := "test-ns"
		app := "test-app"
		cluster := "test-cluster"

		appDir := filepath.Join(tempDir, namespace, app)
		err = os.MkdirAll(appDir, 0755)
		require.NoError(t, err)

		yamlPath := filepath.Join(appDir, fmt.Sprintf("%s.yaml", cluster))
		err = os.WriteFile(yamlPath, []byte("invalid: yaml: content: ["), 0644)
		require.NoError(t, err)

		_, err = UpdateAppVersion(ctx, tempDir, namespace, app, cluster, []Image{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal YAML")
	})
}

func TestUpdateImageTags(t *testing.T) {
	tests := []struct {
		name         string
		yamlContent  string
		newImages    []Image
		expectedTags map[string]string // repository -> expected tag
	}{
		{
			name: "blog app git hash update",
			yamlContent: `controllers:
  main:
    containers:
      main:
        image:
          repository: docker.io/khuedoan/blog
          tag: 6fbd90b77a81e0bcb330fddaa230feff744a7010`,
			newImages: []Image{
				{Repository: "docker.io/khuedoan/blog", Tag: "abc123def456789"},
			},
			expectedTags: map[string]string{
				"docker.io/khuedoan/blog": "abc123def456789",
			},
		},
		{
			name: "actualbudget version update",
			yamlContent: `controllers:
  main:
    containers:
      main:
        image:
          repository: docker.io/actualbudget/actual-server
          tag: 25.6.1-alpine`,
			newImages: []Image{
				{Repository: "docker.io/actualbudget/actual-server", Tag: "25.7.0-alpine"},
			},
			expectedTags: map[string]string{
				"docker.io/actualbudget/actual-server": "25.7.0-alpine",
			},
		},
		{
			name: "mixed registries partial update",
			yamlContent: `controllers:
  main:
    containers:
      main:
        image:
          repository: ghcr.io/silverbulletmd/silverbullet
          tag: v2
  worker:
    containers:
      worker:
        image:
          repository: docker.io/actualbudget/actual-server
          tag: 25.6.1-alpine`,
			newImages: []Image{
				{Repository: "ghcr.io/silverbulletmd/silverbullet", Tag: "v3"},
			},
			expectedTags: map[string]string{
				"ghcr.io/silverbulletmd/silverbullet":  "v3",
				"docker.io/actualbudget/actual-server": "25.6.1-alpine", // unchanged
			},
		},
		{
			name: "local registry with full real structure",
			yamlContent: `defaultPodOptions:
  labels:
    istio.io/dataplane-mode: ambient
controllers:
  main:
    replicas: 2
    strategy: RollingUpdate
    containers:
      main:
        image:
          repository: registry.registry.svc.cluster.local/example-service
          tag: 828c31f942e8913ab2af53a2841c180586c5b7e1
service:
  main:
    controller: main
    ports:
      http:
        port: 8080
        protocol: HTTP`,
			newImages: []Image{
				{Repository: "registry.registry.svc.cluster.local/example-service", Tag: "newgithash12345678901234567890"},
			},
			expectedTags: map[string]string{
				"registry.registry.svc.cluster.local/example-service": "newgithash12345678901234567890",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yamlContent), &node)
			require.NoError(t, err)

			_, err = updateImageTags(&node, tt.newImages)
			require.NoError(t, err)

			// Marshall back to verify changes
			updatedYAML, err := yaml.Marshal(&node)
			require.NoError(t, err)

			var updatedData map[string]interface{}
			err = yaml.Unmarshal(updatedYAML, &updatedData)
			require.NoError(t, err)

			// Verify the expected tag updates
			for expectedRepo, expectedTag := range tt.expectedTags {
				found := false
				findImageTag(updatedData, expectedRepo, expectedTag, &found)
				assert.True(t, found, "Expected to find repository %s with tag %s", expectedRepo, expectedTag)
			}
		})
	}
}

// Helper function to verify image updates in parsed YAML data
func verifyImageUpdates(t *testing.T, data map[string]interface{}, expectedImages []Image) {
	for _, img := range expectedImages {
		found := false
		findImageTag(data, img.Repository, img.Tag, &found)
		assert.True(t, found, "Expected to find repository %s with tag %s", img.Repository, img.Tag)
	}
}

// Recursive helper to find image tags in nested YAML structure
func findImageTag(data interface{}, targetRepo, expectedTag string, found *bool) {
	switch v := data.(type) {
	case map[string]interface{}:
		if imageMap, ok := v["image"].(map[string]interface{}); ok {
			if repo, repoOk := imageMap["repository"].(string); repoOk && repo == targetRepo {
				if tag, tagOk := imageMap["tag"].(string); tagOk && tag == expectedTag {
					*found = true
					return
				}
			}
		}
		for _, value := range v {
			findImageTag(value, targetRepo, expectedTag, found)
		}
	case []interface{}:
		for _, item := range v {
			findImageTag(item, targetRepo, expectedTag, found)
		}
	}
}

func TestUpdateAppVersion_YAMLIndentation(t *testing.T) {
	// Test that YAML is written with 2-space indentation
	tempDir, err := os.MkdirTemp("", "test-yaml-indent-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	namespace := "test"
	app := "indent-test"
	cluster := "local"

	// Create directory structure
	appDir := filepath.Join(tempDir, namespace, app)
	err = os.MkdirAll(appDir, 0755)
	require.NoError(t, err)

	// Create a test YAML file with nested structure
	yamlContent := `controllers:
  main:
    containers:
      main:
        image:
          repository: docker.io/test/app
          tag: v1.0.0
service:
  main:
    controller: main
    ports:
      http:
        port: 8080`

	yamlPath := filepath.Join(appDir, fmt.Sprintf("%s.yaml", cluster))
	err = os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Update with new image
	newImages := []Image{
		{Repository: "docker.io/test/app", Tag: "v2.0.0"},
	}

	ctx := context.Background()
	_, err = UpdateAppVersion(ctx, tempDir, namespace, app, cluster, newImages)
	require.NoError(t, err)

	// Read the updated file and check indentation
	updatedContent, err := os.ReadFile(yamlPath)
	require.NoError(t, err)

	contentStr := string(updatedContent)

	// Check that nested elements use 2-space indentation
	assert.Contains(t, contentStr, "controllers:\n  main:")
	assert.Contains(t, contentStr, "  main:\n    containers:")
	assert.Contains(t, contentStr, "    containers:\n      main:")
	assert.Contains(t, contentStr, "      main:\n        image:")
	assert.Contains(t, contentStr, "        image:\n          repository:")

	// Verify the tag was actually updated
	assert.Contains(t, contentStr, "tag: v2.0.0")
}

package studio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectItemType_Bundle(t *testing.T) {
	// Create a temp directory with a bundle massdriver.yaml
	tmpDir := t.TempDir()
	bundleContent := `
name: test-bundle
description: A test bundle
steps:
  - path: src
    provisioner: opentofu
params:
  properties:
    name:
      type: string
`
	err := os.WriteFile(filepath.Join(tmpDir, "massdriver.yaml"), []byte(bundleContent), 0644)
	require.NoError(t, err)

	itemType, err := DetectItemType(filepath.Join(tmpDir, "massdriver.yaml"))
	require.NoError(t, err)
	assert.Equal(t, ItemTypeBundle, itemType)
}

func TestDetectItemType_ArtifactDefinition(t *testing.T) {
	// Create a temp directory with an artifact definition massdriver.yaml
	tmpDir := t.TempDir()
	artdefContent := `
name: test-artifact
label: Test Artifact
icon: https://example.com/icon.png
schema:
  type: object
  properties:
    id:
      type: string
`
	err := os.WriteFile(filepath.Join(tmpDir, "massdriver.yaml"), []byte(artdefContent), 0644)
	require.NoError(t, err)

	itemType, err := DetectItemType(filepath.Join(tmpDir, "massdriver.yaml"))
	require.NoError(t, err)
	assert.Equal(t, ItemTypeArtifactDefinition, itemType)
}

func TestDetectItemType_LegacyBundle(t *testing.T) {
	// Test legacy bundle detection (has params but no steps)
	tmpDir := t.TempDir()
	bundleContent := `
name: legacy-bundle
params:
  properties:
    name:
      type: string
`
	err := os.WriteFile(filepath.Join(tmpDir, "massdriver.yaml"), []byte(bundleContent), 0644)
	require.NoError(t, err)

	itemType, err := DetectItemType(filepath.Join(tmpDir, "massdriver.yaml"))
	require.NoError(t, err)
	assert.Equal(t, ItemTypeBundle, itemType)
}

func TestScanDirectory(t *testing.T) {
	// Create a temp directory structure
	tmpDir := t.TempDir()

	// Create a bundle
	bundleDir := filepath.Join(tmpDir, "my-bundle")
	err := os.MkdirAll(bundleDir, 0755)
	require.NoError(t, err)

	bundleContent := `
name: my-bundle
steps:
  - path: src
    provisioner: terraform
`
	err = os.WriteFile(filepath.Join(bundleDir, "massdriver.yaml"), []byte(bundleContent), 0644)
	require.NoError(t, err)

	// Create an artifact definition
	artdefDir := filepath.Join(tmpDir, "my-artifact")
	err = os.MkdirAll(artdefDir, 0755)
	require.NoError(t, err)

	artdefContent := `
name: my-artifact
label: My Artifact
schema:
  type: object
`
	err = os.WriteFile(filepath.Join(artdefDir, "massdriver.yaml"), []byte(artdefContent), 0644)
	require.NoError(t, err)

	// Scan the directory
	items, err := ScanDirectory(tmpDir)
	require.NoError(t, err)

	assert.Len(t, items, 2)

	bundles := FilterByType(items, ItemTypeBundle)
	artdefs := FilterByType(items, ItemTypeArtifactDefinition)

	assert.Len(t, bundles, 1)
	assert.Len(t, artdefs, 1)
	assert.Equal(t, "my-bundle", bundles[0].Name)
	assert.Equal(t, "my-artifact", artdefs[0].Name)
}

func TestScanDirectory_SkipsHiddenDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a hidden directory with a bundle
	hiddenDir := filepath.Join(tmpDir, ".hidden")
	err := os.MkdirAll(hiddenDir, 0755)
	require.NoError(t, err)

	bundleContent := `
name: hidden-bundle
steps:
  - path: src
`
	err = os.WriteFile(filepath.Join(hiddenDir, "massdriver.yaml"), []byte(bundleContent), 0644)
	require.NoError(t, err)

	// Scan should find nothing
	items, err := ScanDirectory(tmpDir)
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

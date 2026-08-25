package bundle_test

import (
	"encoding/json"
	"testing"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/oci"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/memory"
)

func TestPackageBundle(t *testing.T) {
	type packageLayer struct {
		MimeType string
	}
	testCases := []struct {
		name           string
		bundleDir      string
		expectedLayers map[string]packageLayer
	}{
		{
			name:      "basic bundle",
			bundleDir: "testdata/publish/simple",
			expectedLayers: map[string]packageLayer{
				"massdriver.yaml": {MimeType: "application/yaml"},
				"operator.md":     {MimeType: "text/markdown"},
				"README.md":       {MimeType: "text/markdown"},
				"src/main.tf":     {MimeType: "application/hcl"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			memStore := memory.New()

			p := oci.Publisher{
				Store: memStore,
			}

			keep, keepErr := bundle.PackageKeep(tc.bundleDir)
			if keepErr != nil {
				t.Fatalf("PackageKeep failed: %v", keepErr)
			}

			tag := "test-tag"
			desc, err := p.Package(t.Context(), tc.bundleDir, tag, bundle.ArtifactType, keep)
			if err != nil {
				t.Fatalf("Package failed: %v", err)
			}

			// Fetch and parse the manifest
			manifestReader, err := memStore.Fetch(t.Context(), desc)
			if err != nil {
				t.Fatalf("failed to fetch manifest: %v", err)
			}

			var manifest ocispec.Manifest
			if err := json.NewDecoder(manifestReader).Decode(&manifest); err != nil {
				t.Fatalf("failed to decode manifest: %v", err)
			}

			for _, layer := range manifest.Layers {
				title := layer.Annotations[ocispec.AnnotationTitle]
				if title == "" {
					t.Fatalf("layer missing title annotation: %v", layer)
				}

				expectedLayer, exists := tc.expectedLayers[title]
				if !exists {
					t.Fatalf("unexpected layer %s found in manifest", title)
					continue
				}

				if layer.MediaType != expectedLayer.MimeType {
					t.Fatalf("expected layer %s to have media type %s, got %s", title, expectedLayer.MimeType, layer.MediaType)
				}
			}

			if len(manifest.Layers) != len(tc.expectedLayers) {
				for title := range tc.expectedLayers {
					found := false
					for _, layer := range manifest.Layers {
						if layer.Annotations[ocispec.AnnotationTitle] == title {
							found = true
							break
						}
					}
					if !found {
						t.Fatalf("expected layer %s not found in manifest", title)
					}
				}
			}
		})
	}
}

package pull_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/commands/bundle/pull"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
)

func TestPull(t *testing.T) {
	ctx := context.Background()

	// Pre-populate source store with fake files
	source := memory.New()
	target := memory.New()
	tag := "latest"

	// Simulate pushing files
	files := map[string]string{
		"massdriver.yaml":       "kind: Bundle\nname: test",
		"schema-artifacts.json": `{"type": "object"}`,
	}

	var layers []ocispec.Descriptor
	for path, data := range files {
		desc := content.NewDescriptorFromBytes("application/octet-stream", []byte(data))
		desc.Annotations = map[string]string{
			ocispec.AnnotationTitle: path,
		}
		if err := source.Push(ctx, desc, bytes.NewReader([]byte(data))); err != nil {
			t.Fatalf("failed to push %s: %v", path, err)
		}
		layers = append(layers, desc)
	}

	// Create and tag manifest
	manifest, err := oras.PackManifest(ctx, source, oras.PackManifestVersion1_1,
		"application/vnd.massdriver.bundle.v1+json", oras.PackManifestOptions{Layers: layers})
	if err != nil {
		t.Fatalf("failed to pack manifest: %v", err)
	}
	if err := source.Tag(ctx, manifest, tag); err != nil {
		t.Fatalf("failed to tag manifest: %v", err)
	}

	tests := []struct {
		name      string
		repo      oras.Target
		target    oras.Target
		tag       string
		wantFiles []string
		wantErr   bool
	}{
		{
			name:   "successful pull",
			repo:   source,
			target: target,
			tag:    tag,
			wantFiles: []string{
				"massdriver.yaml",
				"schema-artifacts.json",
			},
			wantErr: false,
		},
		{
			name:      "missing tag",
			repo:      source,
			target:    memory.New(),
			tag:       "does-not-exist",
			wantFiles: nil,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			puller := &pull.Puller{
				Target: tc.target,
				Repo:   tc.repo,
			}
			desc, pullErr := puller.PullBundle(ctx, tc.tag)
			if (pullErr != nil) != tc.wantErr {
				t.Fatalf("unexpected error = %v, wantErr %v", pullErr, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			// Fetch manifest and verify titles
			rc, err := tc.target.Fetch(ctx, desc)
			if err != nil {
				t.Fatalf("Fetch error: %v", err)
			}
			var manifest ocispec.Manifest
			if err := json.NewDecoder(rc).Decode(&manifest); err != nil {
				t.Fatalf("Manifest decode error: %v", err)
			}

			gotTitles := make(map[string]bool)
			for _, l := range manifest.Layers {
				gotTitles[l.Annotations[ocispec.AnnotationTitle]] = true
			}

			for _, f := range tc.wantFiles {
				if !gotTitles[f] {
					t.Errorf("expected file %q not found in pulled layers", f)
				}
			}
		})
	}
}

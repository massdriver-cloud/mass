package oci_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/massdriver-cloud/mass/internal/oci"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, body := range files {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func fetchManifest(t *testing.T, ctx context.Context, store oras.Target, desc ocispec.Descriptor) ocispec.Manifest {
	t.Helper()
	rc, err := store.Fetch(ctx, desc)
	if err != nil {
		t.Fatalf("fetching manifest: %v", err)
	}
	defer rc.Close()
	var manifest ocispec.Manifest
	if decodeErr := json.NewDecoder(rc).Decode(&manifest); decodeErr != nil {
		t.Fatalf("decoding manifest: %v", decodeErr)
	}
	return manifest
}

func layerTitles(manifest ocispec.Manifest) map[string]ocispec.Descriptor {
	titles := map[string]ocispec.Descriptor{}
	for _, l := range manifest.Layers {
		titles[l.Annotations[ocispec.AnnotationTitle]] = l
	}
	return titles
}

// countingTarget counts pushes of file layers so dedup can be asserted directly.
type countingTarget struct {
	oras.Target
	filePushes int
}

func (c *countingTarget) Push(ctx context.Context, desc ocispec.Descriptor, r io.Reader) error {
	if desc.Annotations[ocispec.AnnotationTitle] != "" {
		c.filePushes++
	}
	return c.Target.Push(ctx, desc, r)
}

func TestPackage(t *testing.T) {
	srcDir := writeTree(t, map[string]string{
		"massdriver.yaml":      "name: aws-s3-bucket\nversion: 1.0.0\n",
		"README.md":            "# readme",
		"instructions/cli.md":  "run it",
		".terraform/junk.tf":   "should not ship",
		"nested/deep/skip.txt": "should not ship",
	})

	keep := func(relPath string) bool {
		switch relPath {
		case "massdriver.yaml", "README.md", "instructions/cli.md":
			return true
		default:
			return false
		}
	}

	store := memory.New()
	publisher := &oci.Publisher{Store: store}

	desc, err := publisher.Package(t.Context(), srcDir, "1.0.0", "application/vnd.massdriver.resource-type.v1+json", keep)
	if err != nil {
		t.Fatalf("Package returned an error: %v", err)
	}

	manifest := fetchManifest(t, t.Context(), store, desc)
	titles := layerTitles(manifest)

	want := []string{"massdriver.yaml", "README.md", "instructions/cli.md"}
	if len(titles) != len(want) {
		t.Errorf("packaged %d layers (%v), want %d", len(titles), titles, len(want))
	}
	for _, w := range want {
		if _, ok := titles[w]; !ok {
			t.Errorf("expected layer %q to be packaged, got %v", w, titles)
		}
	}
	for _, skipped := range []string{".terraform/junk.tf", "nested/deep/skip.txt"} {
		if _, ok := titles[skipped]; ok {
			t.Errorf("layer %q should have been filtered out by the keep predicate", skipped)
		}
	}

	if manifest.ArtifactType != "application/vnd.massdriver.resource-type.v1+json" {
		t.Errorf("ArtifactType = %q, want application/vnd.massdriver.resource-type.v1+json", manifest.ArtifactType)
	}

	// The media type is derived per file from its extension.
	if got := titles["instructions/cli.md"].MediaType; got != "text/markdown" {
		t.Errorf("instructions/cli.md MediaType = %q, want text/markdown", got)
	}
	if got := titles["massdriver.yaml"].MediaType; got != "application/yaml" {
		t.Errorf("massdriver.yaml MediaType = %q, want application/yaml", got)
	}

	resolved, resolveErr := store.Resolve(t.Context(), "1.0.0")
	if resolveErr != nil {
		t.Fatalf("resolving tag: %v", resolveErr)
	}
	if resolved.Digest != desc.Digest {
		t.Errorf("tag 1.0.0 resolves to %s, want %s", resolved.Digest, desc.Digest)
	}
}

func TestPackageNilKeep(t *testing.T) {
	srcDir := writeTree(t, map[string]string{
		"massdriver.yaml": "name: test\n",
		"src/main.tf":     "resource {}",
	})

	store := memory.New()
	publisher := &oci.Publisher{Store: store}

	desc, err := publisher.Package(t.Context(), srcDir, "latest", "application/vnd.massdriver.bundle.v1+json", nil)
	if err != nil {
		t.Fatalf("Package returned an error: %v", err)
	}

	titles := layerTitles(fetchManifest(t, t.Context(), store, desc))
	for _, want := range []string{"massdriver.yaml", "src/main.tf"} {
		if _, ok := titles[want]; !ok {
			t.Errorf("expected layer %q with a nil keep predicate, got %v", want, titles)
		}
	}
}

// Identical bytes are pushed once but still get a layer each, so both paths
// unpack on pull. Breaking this bloats the artifact or drops a file.
func TestPackageDeduplicatesIdenticalContent(t *testing.T) {
	srcDir := writeTree(t, map[string]string{
		"icon.svg":            "<svg/>",
		"instructions/dup.md": "same bytes",
		"instructions/two.md": "same bytes",
	})

	store := &countingTarget{Target: memory.New()}
	publisher := &oci.Publisher{Store: store}

	desc, err := publisher.Package(t.Context(), srcDir, "1.0.0", "application/vnd.massdriver.resource-type.v1+json", nil)
	if err != nil {
		t.Fatalf("Package returned an error: %v", err)
	}

	titles := layerTitles(fetchManifest(t, t.Context(), store, desc))
	if len(titles) != 3 {
		t.Fatalf("got %d distinct layer titles (%v), want 3", len(titles), titles)
	}

	dup, two := titles["instructions/dup.md"], titles["instructions/two.md"]
	if dup.Digest != two.Digest {
		t.Errorf("identical files should share a digest: %s vs %s", dup.Digest, two.Digest)
	}
	if store.filePushes != 2 {
		t.Errorf("pushed %d file blobs, want 2 (the duplicate should be pushed once)", store.filePushes)
	}
}

// Files the mime table doesn't cover still ship, with an empty media type.
func TestPackageExtensionlessFile(t *testing.T) {
	srcDir := writeTree(t, map[string]string{"LICENSE": "MIT"})

	store := memory.New()
	publisher := &oci.Publisher{Store: store}

	desc, err := publisher.Package(t.Context(), srcDir, "1.0.0", "application/vnd.massdriver.bundle.v1+json", nil)
	if err != nil {
		t.Fatalf("Package returned an error: %v", err)
	}

	titles := layerTitles(fetchManifest(t, t.Context(), store, desc))
	if _, ok := titles["LICENSE"]; !ok {
		t.Fatalf("extensionless file was not packaged, got %v", titles)
	}
}

func TestPackageMissingSourceDir(t *testing.T) {
	publisher := &oci.Publisher{Store: memory.New()}
	_, err := publisher.Package(t.Context(), filepath.Join(t.TempDir(), "nope"), "1.0.0", "application/vnd.massdriver.bundle.v1+json", nil)
	if err == nil {
		t.Fatal("expected an error packaging a nonexistent directory, got nil")
	}
}

func TestPublish(t *testing.T) {
	srcDir := writeTree(t, map[string]string{"massdriver.yaml": "name: aws-s3-bucket\n"})

	store, repo := memory.New(), memory.New()
	publisher := &oci.Publisher{Store: store, Repo: repo}

	desc, err := publisher.Package(t.Context(), srcDir, "1.0.0", "application/vnd.massdriver.resource-type.v1+json", nil)
	if err != nil {
		t.Fatalf("Package returned an error: %v", err)
	}
	if publishErr := publisher.Publish(t.Context(), "1.0.0"); publishErr != nil {
		t.Fatalf("Publish returned an error: %v", publishErr)
	}

	resolved, resolveErr := repo.Resolve(t.Context(), "1.0.0")
	if resolveErr != nil {
		t.Fatalf("tag was not published to the repo: %v", resolveErr)
	}
	if resolved.Digest != desc.Digest {
		t.Errorf("repo tag resolves to %s, want %s", resolved.Digest, desc.Digest)
	}

	layer := layerTitles(fetchManifest(t, t.Context(), repo, resolved))["massdriver.yaml"]
	rc, fetchErr := repo.Fetch(t.Context(), layer)
	if fetchErr != nil {
		t.Fatalf("fetching layer from repo: %v", fetchErr)
	}
	defer rc.Close()
	body, _ := io.ReadAll(rc)
	if !bytes.Equal(body, []byte("name: aws-s3-bucket\n")) {
		t.Errorf("layer content = %q, want %q", body, "name: aws-s3-bucket\n")
	}
}

func TestPublishUntaggedManifest(t *testing.T) {
	publisher := &oci.Publisher{Store: memory.New(), Repo: memory.New()}
	if err := publisher.Publish(t.Context(), "1.0.0"); err == nil {
		t.Fatal("expected an error publishing a tag that was never packaged, got nil")
	}
}

func TestMimeTypeFromExtension(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{ext: ".md", want: "text/markdown"},
		{ext: ".yaml", want: "application/yaml"},
		{ext: ".yml", want: "application/yaml"},
		{ext: ".json", want: "application/json"},
		{ext: ".tf", want: "application/hcl"},
		{ext: ".svg", want: "image/svg+xml"},
		{ext: ".png", want: "image/png"},
		{ext: ".xyz", want: ""},
		{ext: "", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.ext, func(t *testing.T) {
			if got := oci.MimeTypeFromExtension(tc.ext); got != tc.want {
				t.Errorf("MimeTypeFromExtension(%q) = %q, want %q", tc.ext, got, tc.want)
			}
		})
	}
}

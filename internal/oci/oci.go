// Package oci contains the raw OCI packaging, publishing, and pulling logic
// shared by bundles and resource types. Callers supply the artifact-type media
// type and a per-file keep predicate; everything else (walking the directory,
// pushing layers, packing the manifest, copying to/from the remote repo) is
// identical across artifact kinds and lives here.
package oci

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
)

// Publisher packages a local directory into an OCI store and pushes it to a
// remote repository.
type Publisher struct {
	Store oras.Target
	Repo  oras.Target
}

// Publish copies the packaged manifest from the local store to the remote
// repository under tag.
func (p *Publisher) Publish(ctx context.Context, tag string) error {
	_, copyErr := oras.Copy(ctx, p.Store, tag, p.Repo, tag, oras.DefaultCopyOptions)
	return copyErr
}

// Package walks srcDir and pushes every file for which keep returns true into
// the store, then packs and tags a manifest of the given artifactType. A nil
// keep predicate includes every file. Paths passed to keep are slash-separated
// and relative to srcDir.
func (p *Publisher) Package(ctx context.Context, srcDir, tag, artifactType string, keep func(relPath string) bool) (ocispec.Descriptor, error) {
	var layers []ocispec.Descriptor
	pushedDigests := make(map[string]string)

	if walkErr := filepath.Walk(srcDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}

		relativePath, relErr := filepath.Rel(srcDir, file)
		if relErr != nil {
			return relErr
		}
		relativePath = filepath.ToSlash(relativePath)

		if keep != nil && !keep(relativePath) {
			return nil
		}

		descriptor, addErr := addFileToStore(ctx, p.Store, file, relativePath, pushedDigests)
		if addErr != nil {
			return addErr
		}
		layers = append(layers, *descriptor)

		return nil
	}); walkErr != nil {
		return ocispec.Descriptor{}, walkErr
	}

	opts := oras.PackManifestOptions{
		Layers: layers,
	}
	manifestDescriptor, packErr := oras.PackManifest(ctx, p.Store, oras.PackManifestVersion1_1, artifactType, opts)
	if packErr != nil {
		return ocispec.Descriptor{}, packErr
	}

	if tagErr := p.Store.Tag(ctx, manifestDescriptor, tag); tagErr != nil {
		return ocispec.Descriptor{}, tagErr
	}

	return manifestDescriptor, nil
}

// Puller copies an artifact from a remote repository into a local target.
type Puller struct {
	Target oras.Target
	Repo   oras.Target
}

// Pull copies the artifact at tag from the remote repository into the target.
func (p *Puller) Pull(ctx context.Context, tag string) (ocispec.Descriptor, error) {
	return oras.Copy(ctx, p.Repo, tag, p.Target, tag, oras.DefaultCopyOptions)
}

func addFileToStore(ctx context.Context, store content.Pusher, filePath, relativePath string, pushedDigests map[string]string) (*ocispec.Descriptor, error) {
	data, readErr := os.ReadFile(filePath)
	if readErr != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, readErr)
	}

	mimeType := MimeTypeFromExtension(filepath.Ext(filePath))
	descriptor := content.NewDescriptorFromBytes(mimeType, data)
	descriptor.Annotations = map[string]string{
		ocispec.AnnotationTitle: relativePath,
	}

	digest := descriptor.Digest.String()
	if _, exists := pushedDigests[digest]; !exists {
		pushErr := store.Push(ctx, descriptor, bytes.NewReader(data))
		if pushErr != nil {
			return nil, fmt.Errorf("pushing %s: %w", filePath, pushErr)
		}
		pushedDigests[digest] = relativePath
	}
	return &descriptor, nil
}

// MimeTypeFromExtension returns the media type for a file extension (including
// the leading dot), or the empty string when unknown.
func MimeTypeFromExtension(ext string) string {
	if mimeType, exists := mimeTypesFromExt[ext]; exists {
		return mimeType
	}
	return ""
}

var mimeTypesFromExt = map[string]string{
	// Text formats
	".txt": "text/plain",
	".md":  "text/markdown",
	".mdx": "text/markdown",
	".csv": "text/csv",
	".log": "text/plain",
	// Configuration / serialization
	".json": "application/json",
	".yaml": "application/yaml",
	".yml":  "application/yaml",
	".toml": "application/toml",
	".ini":  "text/plain", // technically ambiguous
	// HTML, XML
	".html": "text/html",
	".xml":  "application/xml",
	// Source code
	".go":   "text/x-go",
	".py":   "text/x-python",
	".js":   "application/javascript",
	".ts":   "application/typescript",
	".java": "text/x-java-source",
	".rb":   "text/x-ruby",
	".sh":   "application/x-sh",
	".bash": "application/x-sh",
	".c":    "text/x-c",
	".cpp":  "text/x-c++",
	".cs":   "text/x-csharp",
	".php":  "application/x-httpd-php",
	// Infrastructure as code / DevOps
	".tf":         "application/hcl",
	".tfvars":     "application/hcl",
	".hcl":        "application/hcl",
	".rego":       "text/plain", // Open Policy Agent
	".dockerfile": "text/x-dockerfile",
	// Shell scripts / dotfiles
	".env":           "text/plain",
	".gitignore":     "text/plain",
	".gitattributes": "text/plain",
	".bashrc":        "text/x-shellscript",
	// Archives
	".zip":    "application/x-zip-compressed",
	".tar":    "application/x-tar",
	".gz":     "application/x-gzip",
	".tgz":    "application/x-gzip",
	".tar.gz": "application/x-gzip",
	// Binary
	".exe":  "application/vnd.microsoft.portable-executable",
	".dll":  "application/vnd.microsoft.portable-executable",
	".wasm": "application/wasm",
	// Images (commonly used in docs/pipelines)
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".svg":  "image/svg+xml",
	// Certificates / keys
	".pem": "application/x-pem-file",
	".crt": "application/x-x509-ca-cert",
	".key": "application/x-pem-file",
}

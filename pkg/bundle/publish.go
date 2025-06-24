package bundle

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	ignore "github.com/sabhiram/go-gitignore"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
)

type Publisher struct {
	Store oras.Target
	Repo  oras.Target
}

func (p *Publisher) PublishBundle(ctx context.Context, tag string) error {
	_, copyErr := oras.Copy(ctx, p.Store, tag, p.Repo, tag, oras.DefaultCopyOptions)
	return copyErr
}

func (p *Publisher) PackageBundle(ctx context.Context, bundleDir string, tag string) (ocispec.Descriptor, error) {
	ignoreMatcher, ignoreErr := getIgnores(filepath.Join(bundleDir, ".mdignore"))
	if ignoreErr != nil {
		return ocispec.Descriptor{}, ignoreErr
	}

	var layers []ocispec.Descriptor
	var pushedDigests = make(map[string]string)
	if walkErr := filepath.Walk(bundleDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}

		// Calculate relative path from bundle directory
		bundleRelativePath, err := filepath.Rel(bundleDir, file)
		if err != nil {
			return err
		}

		if ignoreMatcher != nil && ignoreMatcher.MatchesPath(bundleRelativePath) {
			return nil
		}

		descriptor, addErr := addFileToStore(ctx, p.Store, file, bundleRelativePath, pushedDigests)
		if addErr != nil {
			return addErr
		}
		layers = append(layers, *descriptor)

		return nil
	}); walkErr != nil {
		return ocispec.Descriptor{}, walkErr
	}

	// 3. Pack the files and tag the packed manifest
	artifactType := "application/vnd.massdriver.bundle.v1+json"
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

func addFileToStore(ctx context.Context, store content.Pusher, filePath string, relativePath string, pushedDigests map[string]string) (*ocispec.Descriptor, error) {
	data, readErr := os.ReadFile(filePath)
	if readErr != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, readErr)
	}

	mimeType := getMimeTypeFromExtension(filepath.Ext(filePath))
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

// Loads patterns from .mdignore file and returns a matcher
func getIgnores(ignorePath string) (*ignore.GitIgnore, error) {
	defaultIgnores := []string{
		// Ignore all files in top level directory except for the following
		"/*",
		"!/massdriver.yaml",
		"!/icon.svg",
		"!/operator.md",
		"!/operator.mdx",
		"!/schema-artifacts.json",
		"!/schema-connections.json",
		"!/schema-params.json",
		"!/schema-ui.json",

		// Do NOT ignore directories (preserve all dirs)
		"!/*/",

		// Ignore all hidden files/directories (e.g., .git, .github, .vscode)
		"/.*",
		"/*/.*",

		// Ignore certain terraform/opentofu files
		"**/.terraform",
		"**/*.tfstate*",
		"**/*.tfvars*",
		// Allow terraform lock files
		"!**/*.terraform.lock.hcl",

		// Allow checkov config file
		"!**/.checkov.yml",
	}

	_, err := os.Stat(ignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ignore.CompileIgnoreLines(defaultIgnores...), nil
		}
		return nil, fmt.Errorf("error checking ignore file: %w", err)
	}

	gi, err := ignore.CompileIgnoreFile(ignorePath)
	if err != nil {
		return nil, fmt.Errorf("invalid ignore file: %w", err)
	}
	return gi, nil
}

func getMimeTypeFromExtension(ext string) string {
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

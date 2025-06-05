package publish

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/massdriver-cloud/mass/pkg/restclient"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	ignore "github.com/sabhiram/go-gitignore"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

func Run(b *bundle.Bundle, c *restclient.MassdriverClient, buildFromDir string) error {
	ctx := context.Background()

	var printBundleName = prettylogs.Underline(b.Name)
	msg := fmt.Sprintf("Publishing %s to package manager", printBundleName)
	fmt.Println(msg)

	store := memory.New()
	ignoreMatcher, ignoreErr := getIgnores(filepath.Join(buildFromDir, ".mdignore"))
	if ignoreErr != nil {
		return fmt.Errorf("getting .mdignore: %w", ignoreErr)
	}

	msg = fmt.Sprintf("Packaging bundle %s for package manager...", printBundleName)
	fmt.Println(msg)

	var layers []ocispec.Descriptor
	if walkErr := filepath.Walk(buildFromDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}

		// Calculate relative path from bundle directory
		bundleRelativePath, err := filepath.Rel(buildFromDir, file)
		if err != nil {
			return err
		}

		if ignoreMatcher != nil && ignoreMatcher.MatchesPath(bundleRelativePath) {
			return nil
		}

		descriptor, addErr := addFileToStore(ctx, store, file, bundleRelativePath)
		if addErr != nil {
			return fmt.Errorf("adding file %s to store: %w", file, addErr)
		}
		layers = append(layers, *descriptor)

		return nil
	}); walkErr != nil {
		return walkErr
	}

	// 3. Pack the files and tag the packed manifest
	artifactType := "application/vnd.massdriver.bundle.v1+json"
	opts := oras.PackManifestOptions{
		Layers: layers,
	}
	manifestDescriptor, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_1, artifactType, opts)
	if err != nil {
		panic(err)
	}

	msg = fmt.Sprintf("Package %s created with digest: %s", printBundleName, manifestDescriptor.Digest)
	fmt.Println(msg)

	tag := "latest"
	if err = store.Tag(ctx, manifestDescriptor, tag); err != nil {
		panic(err)
	}

	msg = fmt.Sprintf("Pushing packaged bundle %s to package manager", printBundleName)
	fmt.Println(msg)

	// 4. Connect to a remote repository
	reg := "2d67-47-229-209-228.ngrok-free.app"
	repo, repoErr := remote.NewRepository(reg + "/sandbox/" + b.Name)
	if repoErr != nil {
		return fmt.Errorf("connecting to remote repository: %w", repoErr)
	}
	// Note: The below code can be omitted if authentication is not required
	repo.Client = &auth.Client{
		Client: retry.DefaultClient,
		Cache:  auth.NewCache(),
		Credential: auth.StaticCredential(reg, auth.Credential{
			Username: "myuser",
			Password: "mypass",
		}),
	}

	// 4. Copy from the file store to the remote repository
	_, copyErr := oras.Copy(ctx, store, tag, repo, tag, oras.DefaultCopyOptions)
	if copyErr != nil {
		return fmt.Errorf("oras copy failed: %w", copyErr)
	}

	msg = fmt.Sprintf("Bundle %s successfully published", printBundleName)
	fmt.Println(msg)

	return nil
}

func addFileToStore(ctx context.Context, store content.Pusher, filePath string, relativePath string) (*ocispec.Descriptor, error) {
	data, readErr := os.ReadFile(filePath)
	if readErr != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, readErr)
	}

	mimeType := getMimeTypeFromExtension(filepath.Ext(filePath))
	descriptor := content.NewDescriptorFromBytes(mimeType, data)
	descriptor.Annotations = map[string]string{
		ocispec.AnnotationTitle: relativePath,
	}

	pushErr := store.Push(ctx, descriptor, bytes.NewReader(data))
	if pushErr != nil {
		return nil, fmt.Errorf("pushing %s: %w", filePath, pushErr)
	}
	return &descriptor, nil
}

// Loads patterns from .mdignore file and returns a matcher
func getIgnores(ignorePath string) (*ignore.GitIgnore, error) {
	defaultIgnores := []string{
		"**/.terraform",
		"*.tfstate*",
		"*.tfvars*",
		".git",
		".github",
		".gitignore",
		".gitlab-ci.yml",
		".vscode",
		".idea",
		".DS_Store",
		"*.md",
		".*",
		"!operator.md",
		"LICENSE",
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

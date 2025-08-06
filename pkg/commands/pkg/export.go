package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/bundle"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	sdkbundle "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/bundle"
	"github.com/mitchellh/mapstructure"
	"oras.land/oras-go/v2/content/file"
)

// Interfaces for dependency injection to enable testing
type FileSystem interface {
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(filename string, data []byte, perm os.FileMode) error
}
type BundleFetcher interface {
	FetchBundle(ctx context.Context, bundleName string, directory string) error
}
type ArtifactDownloader interface {
	DownloadArtifact(ctx context.Context, artifactID string) (string, error)
}
type StateFetcher interface {
	FetchState(ctx context.Context, packageID string, stepPath string) (any, error)
}

type ExportPackageConfig struct {
	Client             *client.Client
	FileSystem         FileSystem
	BundleFetcher      BundleFetcher
	ArtifactDownloader ArtifactDownloader
	StateFetcher       StateFetcher
}

type DefaultFileSystem struct{}

func (dfs *DefaultFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}
func (dfs *DefaultFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

type DefaultBundleFetcher struct {
	Client *client.Client
}

func (dbf *DefaultBundleFetcher) FetchBundle(ctx context.Context, bundleName string, directory string) error {
	repo, repoErr := sdkbundle.GetBundleRepository(dbf.Client, bundleName)
	if repoErr != nil {
		return repoErr
	}

	bundlePath := filepath.Join(directory, "bundle")
	store, fileErr := file.New(bundlePath)
	if fileErr != nil {
		return fileErr
	}
	defer store.Close()

	puller := &bundle.Puller{
		Target: store,
		Repo:   repo,
	}

	_, pullErr := puller.PullBundle(ctx, "latest")
	return pullErr
}

type DefaultArtifactDownloader struct {
	Client *client.Client
}

func (dad *DefaultArtifactDownloader) DownloadArtifact(ctx context.Context, artifactID string) (string, error) {
	return api.DownloadArtifact(ctx, dad.Client, artifactID)
}

type DefaultStateFetcher struct {
	Client *client.Client
}

func (dsf *DefaultStateFetcher) FetchState(ctx context.Context, packageID string, stepPath string) (any, error) {
	var result any
	resp, requestErr := dsf.Client.HTTP.R().
		SetContext(ctx).
		SetResult(&result).
		Get(fmt.Sprintf("/state/%s/%s", packageID, stepPath))

	if requestErr != nil {
		return nil, requestErr
	}
	if resp.IsError() {
		return nil, fmt.Errorf("error fetching state: %s", resp.Status())
	}

	if string(resp.Body()) == `{"version":4}` {
		return nil, nil // No state found, return nil
	}

	return result, nil
}

func RunExport(ctx context.Context, mdClient *client.Client, packageSlugOrID string) error {
	pkg, err := api.GetPackageByName(ctx, mdClient, packageSlugOrID)
	if err != nil {
		return fmt.Errorf("failed to get package %s: %w", packageSlugOrID, err)
	}

	return ExportPackage(ctx, mdClient, pkg, ".")
}

func ExportPackage(ctx context.Context, mdClient *client.Client, pkg *api.Package, baseDirectory string) error {
	config := ExportPackageConfig{
		FileSystem:         &DefaultFileSystem{},
		BundleFetcher:      &DefaultBundleFetcher{Client: mdClient},
		ArtifactDownloader: &DefaultArtifactDownloader{Client: mdClient},
		StateFetcher:       &DefaultStateFetcher{Client: mdClient},
	}

	return ExportPackageWithConfig(ctx, &config, pkg, baseDirectory)
}

// ExportPackageWithConfig is the testable version that accepts dependency injection
func ExportPackageWithConfig(ctx context.Context, config *ExportPackageConfig, pkg *api.Package, baseDirectory string) error {
	validateErr := validatePackageExport(pkg)
	if validateErr != nil {
		return fmt.Errorf("package validation failed: %w", validateErr)
	}

	isRemoteReference := pkg.Status == string(api.PackageStatusExternal)
	isRunning := pkg.Status == string(api.PackageStatusProvisioned)

	if !isRunning && !isRemoteReference {
		fmt.Printf("Package %s is not 'provisioned' or a remote reference, skipping export.\n", pkg.NamePrefix)
		return nil
	}

	directory := filepath.Join(baseDirectory, pkg.Manifest.Slug)
	if err := config.FileSystem.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory for package %s: %w", pkg.NamePrefix, err)
	}

	if isRunning && pkg.Params != nil {
		paramsErr := writeParamsFileWithConfig(config, pkg.Params, directory)
		if paramsErr != nil {
			return fmt.Errorf("failed to write params file for package %s: %w", pkg.NamePrefix, paramsErr)
		}
	}

	if isRunning {
		if err := writeBundleWithConfig(ctx, config, pkg.Manifest.Bundle, pkg.NamePrefix, directory); err != nil {
			return fmt.Errorf("failed to write bundle for package %s: %w", pkg.NamePrefix, err)
		}
	}

	if isRunning && len(pkg.Artifacts) > 0 {
		for _, artifact := range pkg.Artifacts {
			artifactErr := writeArtifactWithConfig(ctx, config, &artifact, directory)
			if artifactErr != nil {
				return fmt.Errorf("failed to write artifact %s for package %s: %w", artifact.Name, pkg.NamePrefix, artifactErr)
			}
		}
	}

	if isRemoteReference && len(pkg.RemoteReferences) > 0 {
		for _, ref := range pkg.RemoteReferences {
			artifactErr := writeArtifactWithConfig(ctx, config, &ref.Artifact, directory)
			if artifactErr != nil {
				return fmt.Errorf("failed to write artifact %s for package %s: %w", ref.Artifact.Field, pkg.NamePrefix, artifactErr)
			}
		}
	}

	if isRunning {
		if err := writeStateWithConfig(ctx, config, pkg, directory); err != nil {
			return fmt.Errorf("failed to write state for package %s: %w", pkg.NamePrefix, err)
		}
	}

	return nil
}

func validatePackageExport(pkg *api.Package) error {
	if pkg == nil {
		return fmt.Errorf("package is nil")
	}

	if pkg.Manifest == nil {
		return fmt.Errorf("package %s manifest is nil", pkg.NamePrefix)
	}

	if pkg.Manifest.Slug == "" {
		return fmt.Errorf("package %s manifest slug is empty", pkg.NamePrefix)
	}

	if pkg.Status == string(api.PackageStatusProvisioned) {
		if pkg.Manifest.Bundle == nil {
			return fmt.Errorf("package %s bundle is nil", pkg.NamePrefix)
		}

		if pkg.Manifest.Bundle.Spec == nil {
			return fmt.Errorf("package %s bundle spec is nil", pkg.NamePrefix)
		}

		if pkg.Manifest.Bundle.Name == "" {
			return fmt.Errorf("package %s bundle name is empty", pkg.NamePrefix)
		}
	}

	if pkg.Status == string(api.PackageStatusExternal) && len(pkg.RemoteReferences) == 0 {
		return fmt.Errorf("package %s is remote reference but has no artifacts", pkg.NamePrefix)
	}

	return nil
}

func writeParamsFileWithConfig(config *ExportPackageConfig, params map[string]any, dir string) error {
	paramsFilePath := filepath.Join(dir, "params.json")

	data, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		return err
	}

	return config.FileSystem.WriteFile(paramsFilePath, data, 0644)
}

func writeBundleWithConfig(ctx context.Context, config *ExportPackageConfig, bun *api.Bundle, pkgNamePrefix string, directory string) error {
	if bun.SpecVersion != "application/vnd.massdriver.bundle.v1+json" {
		fmt.Printf("Bundle %s used by package %s not OCI compliant and cannot be downloaded. Please republish the bundle with an updated CLI to enable downloading.\n", bun.Name, pkgNamePrefix)
		return nil
	}

	return config.BundleFetcher.FetchBundle(ctx, bun.Name, directory)
}

func writeArtifactWithConfig(ctx context.Context, config *ExportPackageConfig, artifact *api.Artifact, directory string) error {
	fileName := fmt.Sprintf("artifact_%s.json", artifact.Field)
	filePath := filepath.Join(directory, fileName)

	data, err := config.ArtifactDownloader.DownloadArtifact(ctx, artifact.ID)
	if err != nil {
		return fmt.Errorf("failed to download artifact %s: %w", artifact.Name, err)
	}

	if err := config.FileSystem.WriteFile(filePath, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write artifact data for %s: %w", artifact.Name, err)
	}

	return nil
}

func writeStateWithConfig(ctx context.Context, config *ExportPackageConfig, pkg *api.Package, directory string) error {
	var unmarshalledBundle bundle.Bundle
	mapstructure.Decode(pkg.Manifest.Bundle.Spec, &unmarshalledBundle)

	var steps []bundle.Step
	if unmarshalledBundle.Steps != nil {
		steps = unmarshalledBundle.Steps
	} else {
		steps = []bundle.Step{
			{
				Path:        "src",
				Provisioner: "terraform",
			},
		}
	}

	for _, step := range steps {
		stateFileName := fmt.Sprintf("%s.tfstate.json", step.Path)
		stateFilePath := filepath.Join(directory, stateFileName)

		result, err := config.StateFetcher.FetchState(ctx, pkg.ID, step.Path)
		if err != nil {
			return fmt.Errorf("failed to fetch state for package %s, step %s: %w", pkg.NamePrefix, step.Path, err)
		}

		if result == nil {
			// no state found, skip writing
			continue
		}

		data, marshalErr := json.MarshalIndent(result, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal terraform state: %w", marshalErr)
		}

		writeErr := config.FileSystem.WriteFile(stateFilePath, data, 0644)
		if writeErr != nil {
			return fmt.Errorf("failed to write state data: %w", writeErr)
		}
	}

	return nil
}

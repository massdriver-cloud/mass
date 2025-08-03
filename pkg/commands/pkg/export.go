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

func RunExport(ctx context.Context, mdClient *client.Client, packageSlugOrID string) error {
	pkg, err := api.GetPackageByName(ctx, mdClient, packageSlugOrID)
	if err != nil {
		return fmt.Errorf("failed to get package %s: %w", packageSlugOrID, err)
	}

	return ExportPackage(ctx, mdClient, pkg, ".")
}

func ExportPackage(ctx context.Context, mdClient *client.Client, pkg *api.Package, baseDirectory string) error {
	validateErr := validatePackageExport(pkg)
	if validateErr != nil {
		return fmt.Errorf("package validation failed: %w", validateErr)
	}

	directory := filepath.Join(baseDirectory, pkg.Manifest.Slug)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory for package %s: %w", pkg.NamePrefix, err)
	}

	isRemoteReference := pkg.Status == string(api.PackageStatusExternal)

	if pkg.Params != nil {
		paramsErr := writeParamsFile(pkg.Params, directory)
		if paramsErr != nil {
			return fmt.Errorf("failed to write params file for package %s: %w", pkg.NamePrefix, paramsErr)
		}
	}

	if !isRemoteReference && pkg.Manifest.Bundle != nil {
		if err := writeBundle(ctx, mdClient, pkg.Manifest.Bundle, directory); err != nil {
			return fmt.Errorf("failed to write bundle for package %s: %w", pkg.NamePrefix, err)
		}
	}

	if !isRemoteReference && len(pkg.Artifacts) > 0 {
		for _, artifact := range pkg.Artifacts {
			artifactErr := writeArtifact(ctx, mdClient, &artifact, directory)
			if artifactErr != nil {
				return fmt.Errorf("failed to write artifact %s for package %s: %w", artifact.Name, pkg.NamePrefix, artifactErr)
			}
		}
	}

	if isRemoteReference && len(pkg.RemoteReferences) > 0 {
		for _, ref := range pkg.RemoteReferences {
			artifactErr := writeArtifact(ctx, mdClient, &ref.Artifact, directory)
			if artifactErr != nil {
				return fmt.Errorf("failed to write artifact %s for package %s: %w", ref.Artifact.Field, pkg.NamePrefix, artifactErr)
			}
		}
	}

	if pkg.Status == string(api.PackageStatusProvisioned) {
		if err := writeState(ctx, mdClient, pkg, directory); err != nil {
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
		return fmt.Errorf("package manifest is nil")
	}

	if pkg.Manifest.Slug == "" {
		return fmt.Errorf("package manifest slug is empty")
	}

	if pkg.Manifest.Bundle == nil {
		return fmt.Errorf("package manifest bundle is nil")
	}

	if pkg.Manifest.Bundle.Spec == nil {
		return fmt.Errorf("package manifest bundle spec is nil")
	}

	if pkg.Manifest.Bundle.Name == "" {
		return fmt.Errorf("package manifest bundle name is empty")
	}

	return nil
}

func writeParamsFile(params map[string]any, dir string) error {
	paramsFilePath := filepath.Join(dir, "params.json")
	file, err := os.Create(paramsFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		return err
	}

	if _, err := file.Write(data); err != nil {
		return err
	}

	return nil
}

func writeBundle(ctx context.Context, mdClient *client.Client, bun *api.Bundle, directory string) error {
	repo, repoErr := sdkbundle.GetBundleRepository(mdClient, bun.Name)
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

	_, pullErr := puller.PullBundle(ctx, "latest") //bun.Version)
	return pullErr
}

func writeArtifact(ctx context.Context, mdClient *client.Client, artifact *api.Artifact, directory string) error {
	fileName := fmt.Sprintf("artifact_%s.json", artifact.Field)
	filePath := filepath.Join(directory, fileName)

	data, err := api.DownloadArtifact(ctx, mdClient, artifact.ID)
	if err != nil {
		return fmt.Errorf("failed to download artifact %s: %w", artifact.Name, err)
	}

	if err := os.WriteFile(filePath, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write artifact data for %s: %w", artifact.Name, err)
	}

	return nil
}

func writeState(ctx context.Context, mdClient *client.Client, pkg *api.Package, directory string) error {
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

		var result any
		resp, err := mdClient.HTTP.R().
			SetContext(ctx).
			SetResult(&result).
			Get(fmt.Sprintf("/state/%s/%s", pkg.ID, step.Path))

		if err != nil {
			return fmt.Errorf("failed to fetch state for package %s, step %s: %w", pkg.NamePrefix, step.Path, err)
		}
		if resp.IsError() {
			return fmt.Errorf("error fetching state for package %s, step %s: %s", pkg.NamePrefix, step.Path, resp.Status())
		}

		file, err := os.Create(stateFilePath)
		if err != nil {
			return fmt.Errorf("failed to create state file: %w", err)
		}
		defer file.Close()

		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal terraform state: %w", err)
		}

		if _, err := file.Write(data); err != nil {
			return fmt.Errorf("failed to write state data: %w", err)
		}
	}

	return nil
}

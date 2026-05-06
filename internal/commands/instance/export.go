package instance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/bundle"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	sdkbundle "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/bundle"
	"oras.land/oras-go/v2/content/file"
)

const emptyStateResponse = `{"version":4}`

// ErrNoState is returned by FetchState when the instance step has no state yet.
var ErrNoState = errors.New("no state found for instance step")

// FileSystem is an interface for dependency injection of filesystem operations to enable testing.
type FileSystem interface {
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(filename string, data []byte, perm os.FileMode) error
}

// BundleFetcher is an interface for downloading a bundle version from a registry into a directory.
type BundleFetcher interface {
	FetchBundle(ctx context.Context, bundleName, version, directory string) error
}

// ResourceLister enumerates the output resources produced by an instance.
type ResourceLister interface {
	ListInstanceResources(ctx context.Context, instanceID string) ([]api.InstanceResource, error)
}

// ResourceExporter retrieves a resource's rendered payload in the requested format.
type ResourceExporter interface {
	ExportResource(ctx context.Context, resourceID, format string) (string, error)
}

// StateFetcher retrieves Terraform/OpenTofu state from a full state backend URL.
type StateFetcher interface {
	FetchState(ctx context.Context, stateURL string) (any, error)
}

// ExportInstanceConfig holds the dependencies needed to export an instance.
type ExportInstanceConfig struct {
	FileSystem       FileSystem
	BundleFetcher    BundleFetcher
	ResourceLister   ResourceLister
	ResourceExporter ResourceExporter
	StateFetcher     StateFetcher
}

// DefaultFileSystem is the production FileSystem implementation backed by the os package.
type DefaultFileSystem struct{}

// MkdirAll creates the directory path and any necessary parents.
func (dfs *DefaultFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// WriteFile writes data to the named file, creating it if necessary.
func (dfs *DefaultFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

// DefaultBundleFetcher is the production BundleFetcher that pulls bundles via OCI.
type DefaultBundleFetcher struct {
	Client *client.Client
}

// FetchBundle downloads the named bundle at the given version into directory using OCI pull.
func (dbf *DefaultBundleFetcher) FetchBundle(ctx context.Context, bundleName, version, directory string) error {
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

	_, pullErr := puller.PullBundle(ctx, version)
	return pullErr
}

// DefaultResourceLister is the production ResourceLister backed by the v2 API.
type DefaultResourceLister struct {
	Client *client.Client
}

// ListInstanceResources returns every output resource produced by the named instance.
func (drl *DefaultResourceLister) ListInstanceResources(ctx context.Context, instanceID string) ([]api.InstanceResource, error) {
	return api.ListInstanceResources(ctx, drl.Client, instanceID)
}

// DefaultResourceExporter is the production ResourceExporter backed by the v2 API.
type DefaultResourceExporter struct {
	Client *client.Client
}

// ExportResource returns the resource's payload rendered in the requested format.
func (dre *DefaultResourceExporter) ExportResource(ctx context.Context, resourceID, format string) (string, error) {
	result, err := api.ExportResource(ctx, dre.Client, resourceID, format)
	if err != nil {
		return "", err
	}
	return result.Rendered, nil
}

// DefaultStateFetcher is the production StateFetcher that retrieves Terraform state by URL.
type DefaultStateFetcher struct {
	Client *client.Client
}

// FetchState retrieves the Terraform state at the given URL using the Massdriver-authenticated HTTP client.
func (dsf *DefaultStateFetcher) FetchState(ctx context.Context, stateURL string) (any, error) {
	var result any
	resp, requestErr := dsf.Client.HTTP.R().
		SetContext(ctx).
		SetResult(&result).
		Get(stateURL)

	if requestErr != nil {
		return nil, requestErr
	}
	if resp.IsError() {
		return nil, fmt.Errorf("error fetching state: %s", resp.Status())
	}

	if string(resp.Body()) == emptyStateResponse {
		return nil, ErrNoState
	}

	return result, nil
}

// RunExport fetches an instance by slug or ID and exports it to the current directory.
func RunExport(ctx context.Context, mdClient *client.Client, instanceSlugOrID string) error {
	inst, err := api.GetInstance(ctx, mdClient, instanceSlugOrID)
	if err != nil {
		return fmt.Errorf("failed to get instance %s: %w", instanceSlugOrID, err)
	}

	return ExportInstance(ctx, mdClient, inst, ".")
}

// ExportInstance exports an instance to baseDirectory using default production dependencies.
func ExportInstance(ctx context.Context, mdClient *client.Client, inst *api.Instance, baseDirectory string) error {
	config := ExportInstanceConfig{
		FileSystem:       &DefaultFileSystem{},
		BundleFetcher:    &DefaultBundleFetcher{Client: mdClient},
		ResourceLister:   &DefaultResourceLister{Client: mdClient},
		ResourceExporter: &DefaultResourceExporter{Client: mdClient},
		StateFetcher:     &DefaultStateFetcher{Client: mdClient},
	}

	return ExportInstanceWithConfig(ctx, &config, inst, baseDirectory)
}

// ExportInstanceWithConfig exports an instance using the provided configuration and dependency overrides.
func ExportInstanceWithConfig(ctx context.Context, config *ExportInstanceConfig, inst *api.Instance, baseDirectory string) error {
	if validateErr := validateInstanceExport(inst); validateErr != nil {
		return fmt.Errorf("instance validation failed: %w", validateErr)
	}

	if inst.Status != "PROVISIONED" {
		fmt.Printf("Instance %s is not 'PROVISIONED', skipping export.\n", inst.ID)
		return nil
	}

	directory := filepath.Join(baseDirectory, inst.Component.ID)
	if err := config.FileSystem.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory for instance %s: %w", inst.ID, err)
	}

	if inst.Params != nil {
		if paramsErr := writeParamsFileWithConfig(config, inst.Params, directory); paramsErr != nil {
			return fmt.Errorf("failed to write params file for instance %s: %w", inst.ID, paramsErr)
		}
	}

	if err := writeBundleWithConfig(ctx, config, inst, directory); err != nil {
		return fmt.Errorf("failed to write bundle for instance %s: %w", inst.ID, err)
	}

	resources, listErr := config.ResourceLister.ListInstanceResources(ctx, inst.ID)
	if listErr != nil {
		return fmt.Errorf("failed to list resources for instance %s: %w", inst.ID, listErr)
	}
	for _, r := range resources {
		if err := writeResourceWithConfig(ctx, config, &r, directory); err != nil {
			return fmt.Errorf("failed to write resource %s for instance %s: %w", r.Resource.Name, inst.ID, err)
		}
	}

	if err := writeStateWithConfig(ctx, config, inst, directory); err != nil {
		return fmt.Errorf("failed to write state for instance %s: %w", inst.ID, err)
	}

	return nil
}

func validateInstanceExport(inst *api.Instance) error {
	if inst == nil {
		return errors.New("instance is nil")
	}

	if inst.Component == nil {
		return fmt.Errorf("instance %s component is nil", inst.ID)
	}

	if inst.Component.ID == "" {
		return fmt.Errorf("instance %s component id is empty", inst.ID)
	}

	if inst.Status == "PROVISIONED" {
		if inst.Bundle == nil {
			return fmt.Errorf("instance %s bundle is nil", inst.ID)
		}

		if inst.Bundle.Name == "" {
			return fmt.Errorf("instance %s bundle name is empty", inst.ID)
		}

		if inst.DeployedVersion == "" {
			return fmt.Errorf("instance %s has no deployed version", inst.ID)
		}
	}

	return nil
}

func writeParamsFileWithConfig(config *ExportInstanceConfig, params map[string]any, dir string) error {
	paramsFilePath := filepath.Join(dir, "params.json")

	data, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		return err
	}

	return config.FileSystem.WriteFile(paramsFilePath, data, 0644)
}

func writeBundleWithConfig(ctx context.Context, config *ExportInstanceConfig, inst *api.Instance, directory string) error {
	return config.BundleFetcher.FetchBundle(ctx, inst.Bundle.Name, inst.DeployedVersion, directory)
}

func writeResourceWithConfig(ctx context.Context, config *ExportInstanceConfig, r *api.InstanceResource, directory string) error {
	fileName := fmt.Sprintf("artifact_%s.json", r.Field)
	filePath := filepath.Join(directory, fileName)

	data, err := config.ResourceExporter.ExportResource(ctx, r.Resource.ID, "json")
	if err != nil {
		return fmt.Errorf("failed to export resource %s: %w", r.Resource.Name, err)
	}

	if err := config.FileSystem.WriteFile(filePath, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write resource data for %s: %w", r.Resource.Name, err)
	}

	return nil
}

func writeStateWithConfig(ctx context.Context, config *ExportInstanceConfig, inst *api.Instance, directory string) error {
	for _, statePath := range inst.StatePaths {
		stateFileName := statePath.StepName + ".tfstate.json"
		stateFilePath := filepath.Join(directory, stateFileName)

		result, err := config.StateFetcher.FetchState(ctx, statePath.StateURL)
		if errors.Is(err, ErrNoState) {
			// no state found, skip writing
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to fetch state for instance %s, step %s: %w", inst.ID, statePath.StepName, err)
		}

		data, marshalErr := json.MarshalIndent(result, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal terraform state: %w", marshalErr)
		}

		if writeErr := config.FileSystem.WriteFile(stateFilePath, data, 0644); writeErr != nil {
			return fmt.Errorf("failed to write state data: %w", writeErr)
		}
	}

	return nil
}

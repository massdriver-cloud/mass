package environment

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/pkg"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func RunExport(ctx context.Context, mdClient *client.Client, environmentIdOrSlug string) error {
	env, getErr := api.GetEnvironment(ctx, mdClient, environmentIdOrSlug)
	if getErr != nil {
		return getErr
	}

	return ExportEnvironment(ctx, mdClient, env, ".")
}

func ExportEnvironment(ctx context.Context, mdClient *client.Client, environment *api.Environment, baseDir string) error {
	validateErr := validateEnvironmentExport(environment)
	if validateErr != nil {
		return fmt.Errorf("environment validation failed: %w", validateErr)
	}

	directory := filepath.Join(baseDir, environment.Slug)
	for _, pack := range environment.Packages {
		exportErr := pkg.ExportPackage(ctx, mdClient, &pack, directory)
		if exportErr != nil {
			return fmt.Errorf("failed to export package %s: %w", pack.NamePrefix, exportErr)
		}
	}

	return nil
}

func validateEnvironmentExport(environment *api.Environment) error {
	if environment == nil {
		return fmt.Errorf("environment cannot be nil")
	}

	if environment.Slug == "" {
		return fmt.Errorf("environment slug is required")
	}

	if len(environment.Packages) == 0 {
		return fmt.Errorf("environment must have at least one package")
	}

	return nil
}

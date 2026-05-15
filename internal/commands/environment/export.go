// Package environment provides commands for managing Massdriver environments.
package environment

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/massdriver-cloud/mass/internal/commands/instance"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/instances"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// RunExport exports all instances in the specified environment to the current directory.
func RunExport(ctx context.Context, mdClient *massdriver.Client, environmentIDOrSlug string) error {
	env, err := mdClient.Environments.Get(ctx, environmentIDOrSlug)
	if err != nil {
		return err
	}

	return ExportEnvironment(ctx, mdClient, env, ".")
}

// ExportEnvironment exports every instance in the given environment into a subdirectory of baseDir.
func ExportEnvironment(ctx context.Context, mdClient *massdriver.Client, env *types.Environment, baseDir string) error {
	if validateErr := validateEnvironmentExport(env); validateErr != nil {
		return fmt.Errorf("environment validation failed: %w", validateErr)
	}

	// Instances aren't embedded on the environment record any more; pull them
	// for this env via the instances service.
	slim, err := mdClient.Instances.List(ctx, instances.ListInput{EnvironmentID: env.ID})
	if err != nil {
		return fmt.Errorf("failed to list instances for environment %s: %w", env.ID, err)
	}
	if len(slim) == 0 {
		return errors.New("environment must have at least one instance")
	}

	directory := filepath.Join(baseDir, env.ID)
	for i := range slim {
		// List returns slim instances (no params/statePaths/resources); fetch
		// the full record so ExportInstance has everything it needs.
		full, getErr := mdClient.Instances.Get(ctx, slim[i].ID)
		if getErr != nil {
			return fmt.Errorf("failed to get instance %s: %w", slim[i].ID, getErr)
		}
		if exportErr := instance.ExportInstance(ctx, mdClient, full, directory); exportErr != nil {
			return fmt.Errorf("failed to export instance %s: %w", full.ID, exportErr)
		}
	}

	return nil
}

func validateEnvironmentExport(env *types.Environment) error {
	if env == nil {
		return errors.New("environment cannot be nil")
	}

	if env.ID == "" {
		return errors.New("environment ID is required")
	}

	return nil
}

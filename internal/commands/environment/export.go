// Package environment provides commands for managing Massdriver environments.
package environment

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/commands/instance"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// RunExport exports all instances in the specified environment to the current directory.
func RunExport(ctx context.Context, mdClient *client.Client, environmentIDOrSlug string) error {
	env, getErr := api.GetEnvironment(ctx, mdClient, environmentIDOrSlug)
	if getErr != nil {
		return getErr
	}

	return ExportEnvironment(ctx, mdClient, env, ".")
}

// ExportEnvironment exports every instance in the given environment into a subdirectory of baseDir.
func ExportEnvironment(ctx context.Context, mdClient *client.Client, env *api.Environment, baseDir string) error {
	if validateErr := validateEnvironmentExport(env); validateErr != nil {
		return fmt.Errorf("environment validation failed: %w", validateErr)
	}

	directory := filepath.Join(baseDir, env.ID)
	for _, inst := range env.Blueprint.Instances {
		exportErr := instance.ExportInstance(ctx, mdClient, &inst, directory)
		if exportErr != nil {
			return fmt.Errorf("failed to export instance %s: %w", inst.ID, exportErr)
		}
	}

	return nil
}

func validateEnvironmentExport(env *api.Environment) error {
	if env == nil {
		return errors.New("environment cannot be nil")
	}

	if env.ID == "" {
		return errors.New("environment ID is required")
	}

	if env.Blueprint == nil || len(env.Blueprint.Instances) == 0 {
		return errors.New("environment must have at least one instance")
	}

	return nil
}

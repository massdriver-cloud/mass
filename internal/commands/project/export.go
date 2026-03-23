// Package project provides commands for managing Massdriver projects.
package project

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/massdriver-cloud/mass/internal/api/v0"
	"github.com/massdriver-cloud/mass/internal/commands/environment"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// RunExport exports all environments and their packages for the specified project.
func RunExport(ctx context.Context, mdClient *client.Client, projectIDOrSlug string) error {
	envs, getErr := api.GetEnvironmentsByProject(ctx, mdClient, projectIDOrSlug)
	if getErr != nil {
		return getErr
	}

	directory := filepath.Join(".", projectIDOrSlug)
	for _, env := range envs {
		exportErr := environment.ExportEnvironment(ctx, mdClient, &env, directory)
		if exportErr != nil {
			return fmt.Errorf("failed to export environment %s: %w", env.Slug, exportErr)
		}
	}

	return nil
}

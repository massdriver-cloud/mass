// Package project provides commands for managing Massdriver projects.
package project

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands/environment"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// RunExport exports all environments and their instances for the specified project.
func RunExport(ctx context.Context, mdClient *client.Client, projectIDOrSlug string) error {
	proj, getErr := api.GetProject(ctx, mdClient, projectIDOrSlug)
	if getErr != nil {
		return getErr
	}

	if len(proj.Components) == 0 {
		fmt.Printf("Project %s has no components to export\n", proj.Name)
		return nil
	}

	envs, listErr := api.ListEnvironments(ctx, mdClient, &api.EnvironmentsFilter{
		ProjectId: &api.IdFilter{Eq: proj.ID},
	})
	if listErr != nil {
		return listErr
	}

	if len(envs) == 0 {
		fmt.Printf("Project %s has no environments to export\n", proj.Name)
		return nil
	}

	directory := filepath.Join(".", proj.ID)
	for _, env := range envs {
		// ListEnvironments returns a summary without the blueprint; fetch the full record for instances.
		full, err := api.GetEnvironment(ctx, mdClient, env.ID)
		if err != nil {
			return fmt.Errorf("failed to get environment %s: %w", env.ID, err)
		}
		if exportErr := environment.ExportEnvironment(ctx, mdClient, full, directory); exportErr != nil {
			return fmt.Errorf("failed to export environment %s: %w", env.ID, exportErr)
		}
	}

	return nil
}

// Package project provides commands for managing Massdriver projects.
package project

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/mass/internal/commands/environment"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
)

// RunExport exports all environments and their instances for the specified project.
func RunExport(ctx context.Context, mdClient *massdriver.Client, projectIDOrSlug string) error {
	proj, err := mdClient.Projects.Get(ctx, projectIDOrSlug)
	if err != nil {
		return err
	}

	if len(proj.Components) == 0 {
		fmt.Printf("Project %s has no components to export\n", proj.Name)
		return nil
	}

	if len(proj.Environments) == 0 {
		fmt.Printf("Project %s has no environments to export\n", proj.Name)
		return nil
	}

	for i := range proj.Environments {
		env := &proj.Environments[i]
		if exportErr := environment.ExportEnvironment(ctx, mdClient, env, proj.ID); exportErr != nil {
			return fmt.Errorf("failed to export environment %s: %w", env.ID, exportErr)
		}
	}

	return nil
}

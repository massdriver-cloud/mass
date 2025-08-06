package project

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/environment"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func RunExport(ctx context.Context, mdClient *client.Client, projectIdOrSlug string) error {
	envs, getErr := api.GetEnvironmentsByProject(ctx, mdClient, projectIdOrSlug)
	if getErr != nil {
		return getErr
	}

	directory := filepath.Join(".", projectIdOrSlug)
	for _, env := range envs {
		exportErr := environment.ExportEnvironment(ctx, mdClient, &env, directory)
		if exportErr != nil {
			return fmt.Errorf("failed to export environment %s: %w", env.Slug, exportErr)
		}
	}

	return nil
}

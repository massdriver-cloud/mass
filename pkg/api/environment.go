package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Khan/genqlient/graphql"
)

type Environment struct {
	ID   string
	Slug string
}

func DeployPreviewEnvironment(client graphql.Client, orgID string, projectID string, credentials []Credential, packageParams map[string]interface{}, ciContext map[string]interface{}) (Environment, error) {
	ctx := context.Background()
	env := Environment{}

	input := PreviewEnvironmentInput{
		Credentials:   credentials,
		PackageParams: packageParams,
		CiContext:     ciContext,
	}

	response, err := deployPreviewEnvironment(ctx, client, orgID, projectID, input)

	if err != nil {
		return env, err
	}

	if response.DeployPreviewEnvironment.Successful {
		return response.toEnvironment(), nil
	}

	msgs, err := json.Marshal(response.DeployPreviewEnvironment.Messages)
	if err != nil {
		return env, fmt.Errorf("failed to deploy preview environment and couldn't marshal error messages: %w", err)
	}

	// TODO: better formatting of errors - custom mutation Error type
	return env, fmt.Errorf("failed to deploy environment: %v", string(msgs))
}

func (r *deployPreviewEnvironmentResponse) toEnvironment() Environment {
	return Environment{
		ID:   r.DeployPreviewEnvironment.Result.Id,
		Slug: r.DeployPreviewEnvironment.Result.Slug,
	}
}

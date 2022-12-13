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
	URL  string
}

const urlTemplate = "https://app.massdriver.cloud/projects/%s/targets/%v"

func DeployPreviewEnvironment(client graphql.Client, orgID string, projectID string, credentials []Credential, packageParams map[string]interface{}, ciContext map[string]interface{}) (*Environment, error) {
	ctx := context.Background()

	input := PreviewEnvironmentInput{
		Credentials:   credentials,
		PackageParams: packageParams,
		CiContext:     ciContext,
	}

	response, err := deployPreviewEnvironment(ctx, client, orgID, projectID, input)

	if err != nil {
		return nil, err
	}

	if response.DeployPreviewEnvironment.Successful {
		return response.DeployPreviewEnvironment.Result.toEnvironment(), nil
	}

	msgs, err := json.Marshal(response.DeployPreviewEnvironment.Messages)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy preview environment and couldn't marshal error messages: %w", err)
	}

	// TODO: better formatting of errors - custom mutation Error type
	return nil, fmt.Errorf("failed to deploy environment: %v", string(msgs))
}

func (e *deployPreviewEnvironmentDeployPreviewEnvironmentTargetPayloadResultTarget) toEnvironment() *Environment {
	return &Environment{
		ID:   e.Id,
		Slug: e.Slug,
		// TODO: use slugs for proj & env once front end has resolved the issues there.
		URL: fmt.Sprintf(urlTemplate, e.Project.Id, e.Id),
	}
}

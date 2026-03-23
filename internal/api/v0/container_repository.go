package api

import (
	"context"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// ContainerRepository holds credentials and URI for a container image repository.
type ContainerRepository struct {
	Token         string
	RepositoryURI string
}

// GetContainerRepository retrieves container repository credentials and URI for the given artifact.
func GetContainerRepository(ctx context.Context, mdClient *client.Client, artifactID, imageName, location string) (*ContainerRepository, error) {
	result := &ContainerRepository{}
	response, err := containerRepository(ctx, mdClient.GQL, mdClient.Config.OrganizationID, artifactID, ContainerRepositoryInput{ImageName: imageName, Location: location})
	if err != nil {
		return result, err
	}

	result.RepositoryURI = response.ContainerRepository.RepoUri
	result.Token = response.ContainerRepository.Token

	return result, nil
}

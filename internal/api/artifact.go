package api

import (
	"context"
	"fmt"

	"github.com/Khan/genqlient/graphql"
)

type Artifact struct {
	Name               string
	ID                 string
	Type               string
	ArtifactDefinition ArtifactDefinition
	Data               interface{}
	Field              string
}

type ArtifactDefinition struct {
	ID   string
	Name string
	Url  string
}

func GetAllArtifacts(client graphql.Client, orgID string) ([]Artifact, error) {
	out, err := getAllArtifactsWithPagination(context.Background(), client, orgID, nil)

	if err != nil {
		return nil, err
	}

	return out, nil
}

func getAllArtifactsWithPagination(ctx context.Context, client graphql.Client, orgID string, cursor *string) ([]Artifact, error) {
	out := make([]Artifact, 0)
	res, err := getAllArtifacts(ctx, client, orgID)

	if err != nil {
		return nil, err
	}

	for _, artifact := range res.Artifacts.Items {
		out = append(out, Artifact{
			Name: artifact.Name,
			ID:   artifact.Id,
			Type: artifact.Type,
			ArtifactDefinition: ArtifactDefinition{
				ID:   artifact.ArtifactDefinition.Id,
				Name: artifact.ArtifactDefinition.Name,
				Url:  artifact.ArtifactDefinition.Url,
			},
			Data: artifact.Data,
		})
	}

	next := &res.Artifacts.Next

	if next != nil && *next != "" {
		nextRes, err := getAllArtifactsWithPagination(ctx, client, orgID, &res.Artifacts.Next)

		if err != nil {
			return nil, err
		}

		out = append(out, nextRes...)
	}

	return out, nil
}

func GetArtifact(client graphql.Client, orgID string, artifactID string) (*Artifact, error) {
	// TODO(amy): The backend needs to permit service accounts to run this query!
	res, err := GetAllArtifacts(client, orgID)

	if err != nil {
		return nil, err
	}

	for _, artifact := range res {
		if artifact.ID == artifactID {
			return &artifact, nil
		}
	}

	return nil, fmt.Errorf("Artifact not found")
}

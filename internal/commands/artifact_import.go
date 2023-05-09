package commands

import (
	"fmt"
	"os"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/spf13/afero"
)

func ArtifactImport(client graphql.Client, OrgID string, fs afero.Fs, artifactName string, artifactType string, artifactFile string) error {
	fmt.Printf("creating artifact %s of type %s\n", artifactName, artifactType)

	bytes, err := os.ReadFile(artifactFile)
	if err != nil {
		return err
	}

	fmt.Println(bytes)

	adInput := api.ArtifactDefinitionInput{Filter: api.ArtifactDefinitionFilters{Service: "AWS"}}
	ads, err := api.GetArtifactDefinitions(client, OrgID, adInput)
	if err != nil {
		return err
	}
	_ = ads

	// for _, ad := range ads {
	// 	fmt.Printf("%v\n", ad.Name)
	// }

	return nil
}

package commands

import (
	"fmt"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/afero"
)

func ArtifactImport(client graphql.Client, OrgID string, fs afero.Fs, artifactName string, artifactType string, artifactFile string) error {
	fmt.Printf("creating artifact %s of type %s\n", artifactName, artifactType)
	return nil
}

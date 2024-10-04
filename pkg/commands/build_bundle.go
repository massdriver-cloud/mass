package commands

import (
	"path"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/provisioners"
	"github.com/massdriver-cloud/mass/pkg/restclient"
)

func BuildBundle(buildPath string, b *bundle.Bundle, c *restclient.MassdriverClient) error {
	err := b.DereferenceSchemas(buildPath, c)
	if err != nil {
		return err
	}

	err = b.WriteSchemas(buildPath)
	if err != nil {
		return err
	}

	combined := b.CombineParamsConnsMetadata()
	for _, step := range b.Steps {
		prov := provisioners.NewProvisioner(step.Provisioner)
		err = prov.ExportMassdriverInputs(path.Join(buildPath, step.Path), combined)
		if err != nil {
			return err
		}
	}

	return nil
}

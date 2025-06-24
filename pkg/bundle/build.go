package bundle

import (
	"path"

	"github.com/massdriver-cloud/mass/pkg/provisioners"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

func (b *Bundle) Build(buildPath string, mdClient *client.Client) error {
	err := b.DereferenceSchemas(buildPath, mdClient)
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

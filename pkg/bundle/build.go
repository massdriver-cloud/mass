// Package bundle provides types and functions for working with Massdriver bundles.
package bundle

import (
	"path/filepath"

	"github.com/massdriver-cloud/mass/pkg/provisioners"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

// Build dereferences schemas, writes them to disk, and exports provisioner inputs for all steps.
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
		err = prov.ExportMassdriverInputs(filepath.Join(buildPath, step.Path), combined)
		if err != nil {
			return err
		}
	}

	return nil
}

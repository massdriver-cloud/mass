package commands

import (
	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/restclient"
	"github.com/massdriver-cloud/mass/internal/terraform"
	"github.com/spf13/afero"
)

func BuildBundle(buildPath string, b *bundle.Bundle, c *restclient.MassdriverClient, fs afero.Fs) error {
	err := b.DereferenceSchemas(buildPath, c, fs)

	if err != nil {
		return err
	}

	err = b.WriteSchemas(buildPath, fs)

	if err != nil {
		return err
	}
	err = terraform.GenerateFiles(buildPath, b, fs)

	if err != nil {
		return err
	}

	return nil
}

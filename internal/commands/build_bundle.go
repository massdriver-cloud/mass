package commands

import (
	"fmt"

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

	for _, step := range stepsOrDefault(b.Steps) {
		switch step.Provisioner {
		case "terraform":
			err = terraform.GenerateFiles(buildPath, step.Path, b, fs)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s is not a supported provisioner", step.Provisioner)
		}
	}

	return nil
}

func stepsOrDefault(steps []bundle.Step) []bundle.Step {
	if steps == nil {
		return []bundle.Step{
			{Path: "src", Provisioner: "terraform"},
		}
	}

	return steps
}

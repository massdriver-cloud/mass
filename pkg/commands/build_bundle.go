package commands

import (
	"fmt"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/provisioners/bicep"
	"github.com/massdriver-cloud/mass/pkg/provisioners/opentofu"
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

	for _, step := range stepsOrDefault(b.Steps) {
		switch step.Provisioner {
		case "terraform", "opentofu":
			err = opentofu.GenerateFiles(buildPath, step.Path, b)
			if err != nil {
				return err
			}
		case "bicep":
			err = bicep.GenerateFiles(buildPath, step.Path, b)
			if err != nil {
				return err
			}
		case "helm":
			continue
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

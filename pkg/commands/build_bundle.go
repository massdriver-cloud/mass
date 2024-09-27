package commands

import (
	"fmt"
	"path"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
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
	for _, step := range stepsOrDefault(b.Steps) {
		prov := provisioners.NewProvisioner(step.Provisioner)
		err = prov.ExportMassdriverInputs(path.Join(buildPath, step.Path), combined)
		if err != nil {
			return err
		}
	}

	return nil
}

func stepsOrDefault(steps []bundle.Step) []bundle.Step {
	if steps == nil {
		msg := fmt.Sprintf(`%s: No steps defined in massdriver.yaml, defaulting to Terraform provisioner. This will be deprecated in a future release. To avoid this warning, please add the following to massdriver.yaml:
steps:
    path: src
    provisioner: terraform`, prettylogs.Orange("Warning"))
		fmt.Println(msg + "\n")
		return []bundle.Step{
			{Path: "src", Provisioner: "terraform"},
		}
	}

	return steps
}

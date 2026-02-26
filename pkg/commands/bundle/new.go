package bundle

import (
	"fmt"
	"path"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/provisioners"
	"github.com/massdriver-cloud/mass/pkg/templates"
)

func RunNew(data *templates.TemplateData) error {
	if err := templates.Render(data); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// if we imported params from existing IaC, pass that to the provisioner in case more initialization should be done
	if data.ExistingParamsPath != "" {
		b, err := bundle.Unmarshal(data.OutputDir)
		if err != nil {
			return fmt.Errorf("failed to unmarshal bundle: %w", err)
		}

		for _, step := range b.Steps {
			prov := provisioners.NewProvisioner(step.Provisioner)
			if err := prov.InitializeStep(path.Join(data.OutputDir, step.Path), data.ExistingParamsPath); err != nil {
				return fmt.Errorf("failed to initialize step %q: %w", step.Path, err)
			}
		}
	}

	return nil
}

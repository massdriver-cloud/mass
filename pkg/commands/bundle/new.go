package bundle

import (
	"fmt"
	"path"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/provisioners"
	"github.com/massdriver-cloud/mass/pkg/templatecache"
)

func RunNew(bundleCache templatecache.TemplateCache, templateData *templatecache.TemplateData) error {
	renderErr := bundleCache.RenderTemplate(templateData)
	if renderErr != nil {
		return fmt.Errorf("failed to render template: %w", renderErr)
	}

	// if we imported params from existing IaC, pass that to the provisioner in case more initialization should be done
	if templateData.ExistingParamsPath != "" {
		b, unmarshalErr := bundle.Unmarshal(templateData.OutputDir)
		if unmarshalErr != nil {
			return fmt.Errorf("failed to unmarshal bundle: %w", unmarshalErr)
		}

		for _, step := range b.Steps {
			prov := provisioners.NewProvisioner(step.Provisioner)
			initErr := prov.InitializeStep(path.Join(templateData.OutputDir, step.Path), templateData.ExistingParamsPath)
			if initErr != nil {
				return fmt.Errorf("failed to initialize step %q: %w", step.Path, initErr)
			}
		}
	}

	return nil
}

package bundle

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/provisioners"
	"github.com/massdriver-cloud/mass/pkg/templates"
)

func RunNew(data *templates.TemplateData) error {
	if data.TemplateName == "" {
		return generateBasicBundle(data)
	}

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

func generateBasicBundle(data *templates.TemplateData) error {
	if err := os.MkdirAll(data.OutputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	content := generateMassdriverYAML(data)

	outputPath := filepath.Join(data.OutputDir, "massdriver.yaml")
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write massdriver.yaml: %w", err)
	}

	return nil
}

func generateMassdriverYAML(data *templates.TemplateData) string {
	yaml := fmt.Sprintf(`schema: draft-07
name: %q
description: %q
source_url: ""
access: private

params:
  required: []
  properties: {}

`, data.Name, data.Description)

	// Add connections
	yaml += "connections:\n"
	if len(data.Connections) == 0 {
		yaml += "  required: []\n  properties: {}\n"
	} else {
		yaml += "  required:\n"
		for _, conn := range data.Connections {
			yaml += fmt.Sprintf("    - %s\n", conn.Name)
		}
		yaml += "  properties:\n"
		for _, conn := range data.Connections {
			yaml += fmt.Sprintf("    %s:\n      $ref: %s\n", conn.Name, conn.ArtifactDefinition)
		}
	}

	yaml += `
artifacts:
  required: []
  properties: {}

ui:
  ui:order:
    - "*"
`

	return yaml
}

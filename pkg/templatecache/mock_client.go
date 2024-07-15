package templatecache

import (
	"fmt"

	"github.com/massdriver-cloud/mass/pkg/mockfilesystem"
)

func NewMockClient(rootTemplateDir string) TemplateCache {
	fetcher := func(filePath string) error {
		directories := []string{
			filePath,
			fmt.Sprintf("%s/massdriver-cloud/application-templates/aws-lambda", filePath),
			fmt.Sprintf("%s/massdriver-cloud/application-templates/aws-vm", filePath),
		}

		err := mockfilesystem.MakeDirectories(directories)

		return err
	}

	return &BundleTemplateCache{
		TemplatePath: rootTemplateDir,
		Fetch:        fetcher,
	}
}

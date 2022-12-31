package templatecache

import (
	"fmt"

	"github.com/massdriver-cloud/mass/internal/mockfilesystem"
	"github.com/spf13/afero"
)

func NewMockClient(rootTemplateDir string, fs afero.Fs) TemplateCache {
	fetcher := func(filePath string) error {
		directories := []string{
			filePath,
			fmt.Sprintf("%s/massdriver-cloud/application-templates/aws-lambda", filePath),
			fmt.Sprintf("%s/massdriver-cloud/application-templates/aws-vm", filePath),
		}

		err := mockfilesystem.MakeDirectories(directories, fs)

		return err
	}

	return &BundleTemplateCache{
		TemplatePath: rootTemplateDir,
		Fetch:        fetcher,
		Fs:           fs,
	}
}

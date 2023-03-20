package publish

import (
	"bytes"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/restclient"
	"github.com/spf13/afero"
)

func Run(b *bundle.Bundle, c *restclient.MassdriverClient, fs afero.Fs, buildFromDir string) error {
	publisher := &Publisher{
		Bundle:     b,
		RestClient: c,
		Fs:         fs,
		BuildDir:   buildFromDir,
	}

	s3SignedURL, err := publisher.SubmitBundle()

	if err != nil {
		return err
	}

	var buf bytes.Buffer

	if err = publisher.ArchiveBundle(&buf); err != nil {
		return err
	}

	if err = publisher.PushArchiveToPackageManager(s3SignedURL, &buf); err != nil {
		return err
	}

	return nil
}

package publish

import (
	"bytes"
	"fmt"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/prettylogs"
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

	var bundleName = prettylogs.Underline(b.Name)
	var access = prettylogs.Underline(b.Access)
	msg := fmt.Sprintf("Publishing %s with %s visibility", bundleName, access)
	fmt.Println(msg)

	s3SignedURL, err := publisher.SubmitBundle()

	if err != nil {
		fmt.Println(err)
		return err
	}

	msg = fmt.Sprintf("%s published successfully to with %s visibility", bundleName, access)
	fmt.Println(msg)

	var buf bytes.Buffer

	msg = fmt.Sprintf("Packaging bundle %s for package manager", bundleName)
	fmt.Println(msg)
	if err = publisher.ArchiveBundle(&buf); err != nil {
		fmt.Println(err)
		return err
	}

	msg = fmt.Sprintf("Package %s created", bundleName)
	fmt.Println(msg)

	msg = fmt.Sprintf("Pushing packaged bundle %s to package manager", bundleName)
	fmt.Println(msg)
	if err = publisher.PushArchiveToPackageManager(s3SignedURL, &buf); err != nil {
		fmt.Println(err)
		return err
	}

	msg = fmt.Sprintf("Bundle %s successfully published", bundleName)
	fmt.Println(msg)

	return nil
}

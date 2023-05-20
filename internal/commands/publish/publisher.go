package publish

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/restclient"
	"github.com/spf13/afero"
)

const (
	MassdriverURL                 = "https://api.massdriver.cloud/"
	MassdriverYamlFilename        = "massdriver.yaml"
	ArtifactsSchemaFilename       = "schema-artifacts.json"
	ConnectionsSchemaFilename     = "schema-connections.json"
	ParamsSchemaFilename          = "schema-params.json"
	UISchemaFilename              = "schema-ui.json"
	DevParamsFilename             = "_params.auto.tfvars.json"
	MaxBundleSizeMB               = 10
	MaxFileSizeMB                 = 1
	PackageManagerDirectoryPrefix = "bundle"
)

type Publisher struct {
	Bundle     *bundle.Bundle
	RestClient *restclient.MassdriverClient
	Fs         afero.Fs
	BuildDir   string
}

type S3PresignEndpointResponse struct {
	Error                 xml.Name `xml:"Error"`
	Code                  string   `xml:"Code"`
	Message               string   `xml:"Message"`
	AWSAccessKeyID        string   `xml:"AWSAccessKeyId"`
	StringToSign          string   `xml:"StringToSign"`
	SignatureProvided     string   `xml:"SignatureProvided"`
	StringToSignBytes     []byte   `xml:"StringToSignBytes"`
	CanonicalRequest      string   `xml:"CanonicalRequest"`
	CanonicalRequestBytes []byte   `xml:"CanonicalRequestBytes"`
	RequestID             string   `xml:"RequestId"`
	HostID                string   `xml:"HostId"`
}

var fileIgnores = []string{
	".terraform",
	".tfstate",
	".tfvars",
	".md",
	".git",
	".DS_Store",
}

var fileAllows = []string{
	MassdriverYamlFilename,
	ArtifactsSchemaFilename,
	ConnectionsSchemaFilename,
	ParamsSchemaFilename,
	UISchemaFilename,
	"src",
}

func (p *Publisher) SubmitBundle() (string, error) {
	//TODO: Add log message for publish and response
	body, err := p.Bundle.GenerateBundlePublishBody(p.BuildDir, p.Fs)

	if err != nil {
		return "", err
	}

	return p.RestClient.PublishBundle(body)
}

func (p *Publisher) ArchiveBundle(buf io.Writer) error {
	allowList := getAllowList(p.Bundle)
	copyConfig := CopyConfig{
		Allows:  allowList,
		Ignores: fileIgnores,
	}

	gzipWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(gzipWriter)

	packager := newPackager(&copyConfig, p.Fs)
	errCompress := packager.createArchiveWithFilter(p.BuildDir, PackageManagerDirectoryPrefix, tarWriter)

	if errCompress != nil {
		return errCompress
	}

	// produce tar
	err := tarWriter.Close()

	if err != nil {
		return err
	}

	// produce gzip
	err = gzipWriter.Close()

	if err != nil {
		return err
	}

	return nil
}

func (p Publisher) PushArchiveToPackageManager(url string, object io.Reader) error {
	// TODO: Add log message for push to s3
	req, err := http.NewRequestWithContext(context.Background(), "PUT", url, object)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var respContent S3PresignEndpointResponse
		var respBody bytes.Buffer
		if _, readErr := respBody.ReadFrom(resp.Body); readErr != nil {
			return readErr
		}
		if xmlErr := xml.Unmarshal(respBody.Bytes(), &respContent); xmlErr != nil {
			return fmt.Errorf("enountered non-200 response code, unable to unmarshal xml response body: %v: original error: %w", respBody.String(), xmlErr)
		}

		return errors.New("unable to upload content: " + respContent.Message)
	}

	return nil
}

func getAllowList(b *bundle.Bundle) []string {
	allAllows := []string{}
	allAllows = append(allAllows, fileAllows...)

	if b.Steps != nil {
		for _, step := range b.Steps {
			allAllows = append(allAllows, step.Path)
		}
	}

	return removeDuplicateValues(allAllows)
}

func removeDuplicateValues(stringSlice []string) []string {
	keysSeen := make(map[string]bool)
	list := []string{}

	for _, entry := range stringSlice {
		if !keysSeen[entry] {
			list = append(list, entry)
			keysSeen[entry] = true
		}
	}

	return list
}

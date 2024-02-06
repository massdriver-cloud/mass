package image

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/moby/moby/pkg/jsonmessage"
	"github.com/moby/term"
)

const AWS = "AWS"
const GCP = "GCP"
const AZURE = "Azure"

func Push(client graphql.Client, input PushImageInput, imageClient Client) error {
	var imageName = prettylogs.Underline(input.ImageName)
	var location = prettylogs.Underline(input.Location)

	msg := fmt.Sprintf("Creating repository for image %s in region %s and fetching single use credentials", imageName, location)
	fmt.Println(msg)

	containerRepository, err := api.GetContainerRepository(client, input.ArtifactID, input.OrganizationID, input.ImageName, input.Location)

	if err != nil {
		return err
	}

	cloudName := identifyCloudByRepositoryURI(containerRepository.RepositoryURI)

	var logCloud = prettylogs.Underline(cloudName)
	msg = fmt.Sprintf("%s credentials fetched successfully", logCloud)
	fmt.Println(msg)

	if !input.SkipBuild {
		err = imageClient.BuildImage(input, containerRepository)
		if err != nil {
			return err
		}
	}

	return imageClient.PushImage(input, containerRepository)
}
func identifyCloudByRepositoryURI(uri string) string {
	switch {
	case strings.Contains(uri, "amazonaws.com"):
		return AWS
	case strings.Contains(uri, "azurecr.io"):
		return AZURE
	case strings.Contains(uri, "docker.pkg.dev"):
		return GCP
	default:
		return "unknown"
	}
}

func handleResponseBuffer(buf io.ReadCloser) error {
	defer buf.Close()

	return printDockerOutput(buf)
}

func printDockerOutput(rd io.Reader) error {
	fd, isTerminal := term.GetFdInfo(os.Stdout)
	return jsonmessage.DisplayJSONMessagesStream(rd, os.Stdout, fd, isTerminal, nil)
}

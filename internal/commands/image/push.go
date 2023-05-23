package image

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/prettylogs"
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

	var logTag = prettylogs.Underline(input.Tag)

	msg = fmt.Sprintf("Building %s and tagging the image with %s", imageName, logTag)
	fmt.Println(msg)

	res, err := imageClient.BuildImage(input, containerRepository)

	if err != nil {
		return err
	}

	err = handleResponseBuffer(res.Body)

	if err != nil {
		return err
	}

	fmt.Println("Pushing image to repository. This may take a few minutes")

	rd, err := imageClient.PushImage(input, containerRepository)

	if err != nil {
		return err
	}

	err = handleResponseBuffer(rd)

	if err != nil {
		return err
	}

	var fqn = prettylogs.Underline(imageFqn(containerRepository.RepositoryURI, input.ImageName, input.Tag))
	msg = fmt.Sprintf("Image %s pushed successfully", fqn)
	fmt.Println(msg)

	return nil
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
	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
	}

	errLine := &ErrorLine{}

	_ = json.Unmarshal([]byte(lastLine), errLine)

	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	err := scanner.Err()

	if err != nil {
		return err
	}

	return nil
}

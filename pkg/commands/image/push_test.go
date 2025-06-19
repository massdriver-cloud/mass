package image_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/image"

	"github.com/Khan/genqlient/graphql"
	"github.com/docker/docker/api/types"
	dockerImage "github.com/docker/docker/api/types/image"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
)

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func NopCloser(r io.Reader) io.ReadCloser {
	return nopCloser{r}
}

type mockCli struct{}

func (mockCli) ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error) {
	buffer := bytes.NewBuffer(make([]byte, 0))
	return types.ImageBuildResponse{Body: NopCloser(buffer)}, nil
}

func (mockCli) ImagePush(ctx context.Context, image string, options dockerImage.PushOptions) (io.ReadCloser, error) {
	return NopCloser(bytes.NewBuffer(make([]byte, 0))), nil
}

func (mockCli) ImageTag(ctx context.Context, source, target string) error {
	return nil
}

type mockGQLClient struct{}

func (mockGQLClient) MakeRequest(ctx context.Context, req *graphql.Request, resp *graphql.Response) error {
	buffer := bytes.NewBufferString("{\"data\": {\"containerRepository\": {\"token\": \"bogustoken\", \"repoUri\": \"0000000.ecr.dkr.amazonaws.com\"}}}")
	err := json.NewDecoder(buffer).Decode(resp)

	return err
}

func (mockGQLClient) GetContainerRepository(client graphql.Client, artifactID string, orgID string, imageName string, location string) (*api.ContainerRepository, error) {
	return &api.ContainerRepository{
		Token:         "bogustoken",
		RepositoryURI: "https://0000000.ecr.dkr.amazonaws.com",
	}, nil
}

func TestPushLatestImage(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput((os.Stderr))
	}()

	mdClient := client.Client{
		GQL: &mockGQLClient{},
	}
	imageClient := image.Client{
		Cli: &mockCli{},
	}
	input := image.PushImageInput{
		ImageName:          "test/docker",
		Location:           "us-west-2",
		ArtifactID:         "00000-000-000-00000000",
		OrganizationID:     "00000-000-000-00000000",
		Tags:               []string{"latest"},
		DockerBuildContext: ".",
		Dockerfile:         "DockerFile",
	}

	err := image.Push(t.Context(), &mdClient, input, imageClient)

	if err != nil {
		t.Fatal(err)
	}
}

func TestPushImage(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput((os.Stderr))
	}()

	mdClient := client.Client{
		GQL: &mockGQLClient{},
	}
	imageClient := image.Client{
		Cli: &mockCli{},
	}
	input := image.PushImageInput{
		ImageName:          "test/docker",
		Location:           "us-west-2",
		ArtifactID:         "00000-000-000-00000000",
		OrganizationID:     "00000-000-000-00000000",
		Tags:               []string{"some-tag"},
		DockerBuildContext: ".",
		Dockerfile:         "DockerFile",
	}

	err := image.Push(t.Context(), &mdClient, input, imageClient)

	if err != nil {
		t.Fatal(err)
	}
}

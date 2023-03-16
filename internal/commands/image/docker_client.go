package image

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/massdriver-cloud/mass/internal/api"
)

type DockerClient interface {
	ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error)
	ImagePush(ctx context.Context, image string, options types.ImagePushOptions) (io.ReadCloser, error)
}

type Client struct {
	Cli DockerClient
}

func NewImageClient() (Client, error) {
	cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithVersion("1.41"))

	if err != nil {
		return Client{}, errors.New("docker Engine API is not installed. to install it go to https://docs.docker.com/get-docker/ and follow the instructions")
	}

	return Client{Cli: cli}, nil
}

func (c *Client) BuildImage(input PushImageInput, containerRepository *api.ContainerRepository) (*types.ImageBuildResponse, error) {
	tar, err := packageBuildDirectory(input.DockerBuildContext)

	if err != nil {
		return nil, err
	}

	opts := types.ImageBuildOptions{
		Dockerfile: input.Dockerfile,
		Tags:       []string{imageFqn(containerRepository.RepositoryURI, input.ImageName, input.Tag)},
		Remove:     true,
		Platform:   input.TargetPlatform,
	}

	ctx := context.Background()

	res, err := c.Cli.ImageBuild(ctx, tar, opts)
	return &res, err
}

func (c *Client) PushImage(input PushImageInput, containerRepository *api.ContainerRepository) (io.ReadCloser, error) {
	ctx := context.Background()
	auth, err := createAuthForCloud(containerRepository)

	if err != nil {
		return nil, err
	}

	res, err := c.Cli.ImagePush(ctx, imageFqn(containerRepository.RepositoryURI, input.ImageName, input.Tag), types.ImagePushOptions{RegistryAuth: auth})

	return res, err
}

func packageBuildDirectory(buildContext string) (io.ReadCloser, error) {
	return archive.TarWithOptions(buildContext, &archive.TarOptions{})
}

func imageFqn(uri, imageName, tag string) string {
	return fmt.Sprintf("%s/%s:%s", repoPrefix(uri), imageName, tag)
}

func repoPrefix(uri string) string {
	return strings.ReplaceAll(uri, "https://", "")
}

func createAuthForCloud(containerRepository *api.ContainerRepository) (string, error) {
	authConfig := &types.AuthConfig{}

	err := setAuthUserNameByCloud(containerRepository, authConfig)

	if err != nil {
		return "", err
	}

	err = maybeRemoveSuffix(containerRepository, authConfig)

	if err != nil {
		return "", err
	}

	authConfig.Password = containerRepository.Token

	authConfigBytes, err := json.Marshal(authConfig)

	if err != nil {
		return "", err
	}

	encodedAuth := base64.URLEncoding.EncodeToString(authConfigBytes)

	return encodedAuth, nil
}

func setAuthUserNameByCloud(containerRepository *api.ContainerRepository, auth *types.AuthConfig) error {
	switch identifyCloudByRepositoryURI(containerRepository.RepositoryURI) {
	case AWS:
		auth.Username = "AWS"
	case AZURE:
		auth.Username = "00000000-0000-0000-0000-000000000000"
	case GCP:
		auth.Username = "oauth2accesstoken"
	default:
		return fmt.Errorf("container repositories are not supported for %s", containerRepository.RepositoryURI)
	}

	return nil
}

func maybeRemoveSuffix(containerRepository *api.ContainerRepository, auth *types.AuthConfig) error {
	r := regexp.MustCompile("[a-zA-Z0-9-_]+.docker.pkg.dev")

	switch identifyCloudByRepositoryURI(containerRepository.RepositoryURI) {
	case GCP:
		auth.ServerAddress = r.FindString(containerRepository.RepositoryURI)
		return nil
	case AWS:
		auth.ServerAddress = containerRepository.RepositoryURI
		return nil
	case AZURE:
		auth.ServerAddress = containerRepository.RepositoryURI
		return nil
	default:
		return fmt.Errorf("container repositories are not supported for %s", containerRepository.RepositoryURI)
	}
}

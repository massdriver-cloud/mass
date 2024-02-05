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
	"github.com/docker/docker/api/types/registry"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
)

var dockerURIPattern = regexp.MustCompile("[a-zA-Z0-9-_]+.docker.pkg.dev")

type DockerClient interface {
	ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error)
	ImagePush(ctx context.Context, image string, options types.ImagePushOptions) (io.ReadCloser, error)
}

type Client struct {
	Cli DockerClient
}

func NewImageClient() (Client, error) {
	cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithAPIVersionNegotiation())

	if err != nil {
		return Client{}, errors.New("docker Engine API is not installed. to install it go to https://docs.docker.com/get-docker/ and follow the instructions")
	}

	return Client{Cli: cli}, nil
}

func (c *Client) BuildImage(input PushImageInput, containerRepository *api.ContainerRepository) error {
	tar, err := packageBuildDirectory(input.DockerBuildContext)
	if err != nil {
		return err
	}

	var imageFqns []string
	for _, tag := range input.Tags {
		imageFqns = append(imageFqns, imageFqn(containerRepository.RepositoryURI, input.ImageName, tag))
	}

	logTag := prettylogs.Underline(strings.Join(imageFqns, "\n"))
	fmt.Printf("Building images: \n%s\n", logTag)

	opts := types.ImageBuildOptions{
		Dockerfile: input.Dockerfile,
		Tags:       imageFqns,
		Remove:     true,
		Platform:   input.TargetPlatform,
	}

	if input.UseBuildKit {
		opts.Version = types.BuilderBuildKit
	}

	if input.CacheFrom != "" {
		opts.CacheFrom = []string{input.CacheFrom}
	}

	ctx := context.Background()

	res, err := c.Cli.ImageBuild(ctx, tar, opts)
	if err != nil {
		return fmt.Errorf("ImageBuild: %w", err)
	}
	err = handleResponseBuffer(res.Body)
	if err != nil {
		return fmt.Errorf("HandleRespone: %w", err)
	}

	return nil
}

func (c *Client) PushImage(input PushImageInput, containerRepository *api.ContainerRepository) error {
	ctx := context.Background()
	auth, authErr := createAuthForCloud(containerRepository)
	if authErr != nil {
		return authErr
	}

	for _, tag := range input.Tags {
		res, err := c.Cli.ImagePush(ctx, imageFqn(containerRepository.RepositoryURI, input.ImageName, tag), types.ImagePushOptions{RegistryAuth: auth})
		if err != nil {
			return err
		}
		err = handleResponseBuffer(res)
		if err != nil {
			return err
		}
		fqn := prettylogs.Underline(imageFqn(containerRepository.RepositoryURI, input.ImageName, tag))
		msg := fmt.Sprintf("Image %s pushed successfully", fqn)
		fmt.Println(msg)
	}

	return nil
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
	authConfig := &registry.AuthConfig{}

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

func setAuthUserNameByCloud(containerRepository *api.ContainerRepository, auth *registry.AuthConfig) error {
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

func maybeRemoveSuffix(containerRepository *api.ContainerRepository, auth *registry.AuthConfig) error {
	switch identifyCloudByRepositoryURI(containerRepository.RepositoryURI) {
	case GCP:
		auth.ServerAddress = dockerURIPattern.FindString(containerRepository.RepositoryURI)
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

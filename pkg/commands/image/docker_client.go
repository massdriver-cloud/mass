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
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
)

var dockerURIPattern = regexp.MustCompile("[a-zA-Z0-9-_]+.docker.pkg.dev")

type DockerClient interface {
	ImageBuild(ctx context.Context, buildContext io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error)
	ImagePush(ctx context.Context, image string, options image.PushOptions) (io.ReadCloser, error)
	ImageTag(ctx context.Context, source, target string) error
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

func (c *Client) BuildImage(input PushImageInput, containerRepository *api.ContainerRepository) error {
	tar, err := packageBuildDirectory(input.DockerBuildContext)
	if err != nil {
		return err
	}

	var imageFqns []string
	for _, tag := range input.Tags {
		imageFqns = append(imageFqns, getImageFQN(containerRepository.RepositoryURI, input.ImageName, tag))
	}

	logTag := prettylogs.Underline(strings.Join(imageFqns, "\n"))
	fmt.Printf("Building images: \n%s\n", logTag)

	opts := types.ImageBuildOptions{
		Dockerfile: input.Dockerfile,
		Tags:       imageFqns,
		Remove:     true,
		Platform:   input.TargetPlatform,
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
		imageFQN := getImageFQN(containerRepository.RepositoryURI, input.ImageName, tag)

		// Check the image name for the registry URL, if it doesn't have it yet tag the image with the FQN
		// image-namespace/image:latest > registry-url/image-namespace/image:latest
		if input.SkipBuild && !strings.HasPrefix(input.ImageName, dropRepoPrefix(containerRepository.RepositoryURI)) {
			err := c.tagImageWithFQN(ctx, fmt.Sprintf("%s:%s", input.ImageName, tag), imageFQN)
			if err != nil {
				return err
			}
		}

		fmt.Println("Pushing image to repository. This may take a few minutes")
		res, err := c.Cli.ImagePush(ctx, imageFQN, image.PushOptions{RegistryAuth: auth})
		if err != nil {
			return err
		}
		err = handleResponseBuffer(res)
		if err != nil {
			return err
		}
		fqn := prettylogs.Underline(imageFQN)
		msg := fmt.Sprintf("Image %s pushed successfully", fqn)
		fmt.Println(msg)
	}

	return nil
}

func (c *Client) tagImageWithFQN(ctx context.Context, current, fullName string) error {
	fmt.Printf("Tagging image %s with %s\n", prettylogs.Underline(current), prettylogs.Underline(fullName))
	return c.Cli.ImageTag(ctx, current, fullName)
}

func packageBuildDirectory(buildContext string) (io.ReadCloser, error) {
	return archive.TarWithOptions(buildContext, &archive.TarOptions{})
}

func getImageFQN(uri, imageName, tag string) string {
	return fmt.Sprintf("%s/%s:%s", dropRepoPrefix(uri), imageName, tag)
}

func dropRepoPrefix(uri string) string {
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

package cmd

import (
	"fmt"
	"strings"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/commands/image"
	"github.com/massdriver-cloud/mass/pkg/config"
	"github.com/spf13/cobra"
)

type PushFlag struct {
	Flag      string
	Attribute *string
}

var pushInput = image.PushImageInput{}

func NewCmdImage() *cobra.Command {
	imageCmd := &cobra.Command{
		Use:   "image",
		Short: "Container image integration Massdriver",
	}

	imagePushCmd := &cobra.Command{
		Use:   "push <namespace>/<image-name>",
		Short: "Push an image to ECR, ACR or GAR",
		Long:  helpdocs.MustRender("image/push"),
		RunE:  runImagePush,
		Args:  cobra.ExactArgs(1),
	}
	imagePushCmd.Flags().StringVarP(&pushInput.DockerBuildContext, "build-context", "b", ".", "Path to the directory to build the image from")
	imagePushCmd.Flags().StringVarP(&pushInput.Dockerfile, "dockerfile", "f", "Dockerfile", "Name of the dockerfile to build from if you have named it anything other than Dockerfile")
	imagePushCmd.Flags().StringSliceVarP(&pushInput.Tags, "image-tag", "t", []string{"latest"}, "Unique identifier for this version of the image")
	imagePushCmd.Flags().StringVarP(&pushInput.ArtifactID, "artifact", "a", "", "Massdriver ID of the artifact used to create the repository and generate repository credentials")
	_ = imagePushCmd.MarkFlagRequired("artifact")
	imagePushCmd.Flags().StringVarP(&pushInput.Location, "region", "r", "", "Cloud region to push the image to")
	_ = imagePushCmd.MarkFlagRequired("region")
	imagePushCmd.Flags().StringVarP(&pushInput.TargetPlatform, "platform", "p", "linux/amd64", "Set platform if server is multi-platform capable")
	imagePushCmd.Flags().StringVarP(&pushInput.CacheFrom, "cache-from", "c", "", "Path to image used for caching")
	imagePushCmd.Flags().BoolVarP(&pushInput.SkipBuild, "skip-build", "s", false, "Skip building the image before pushing")

	imageCmd.AddCommand(imagePushCmd)

	return imageCmd
}

func runImagePush(cmd *cobra.Command, args []string) error {
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}

	if !strings.Contains(config.URL, "https://api.massdriver.cloud") {
		return fmt.Errorf("image management is only supported in the Massdriver Cloud. Your current API URL is %s, which is not supported for image management", config.URL)
	}

	pushInput.OrganizationID = config.OrgID
	pushInput.ImageName = args[0]

	err := validatePushInputAndAddFlags(&pushInput)
	if err != nil {
		return err
	}

	gqlclient := api.NewClient(config.URL, config.APIKey)
	imageClient, err := image.NewImageClient()

	if err != nil {
		return err
	}

	return image.Push(gqlclient, pushInput, imageClient)
}

func validatePushInputAndAddFlags(input *image.PushImageInput) error {
	if invalidImageName(input.ImageName) {
		return fmt.Errorf("%s is an invalid image name. Massdriver enforces the practice of namespacing images. please enter an image name in the format of namespace/image-name", input.ImageName)
	}
	return nil
}

func invalidImageName(imageName string) bool {
	return !(len(strings.Split(imageName, "/")) == 2)
}

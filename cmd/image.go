package cmd

import (
	"fmt"
	"strings"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/commands/image"
	"github.com/massdriver-cloud/mass/internal/config"
	"github.com/spf13/cobra"
)

var imagePushCmdHelp = mustRenderHelpDoc("image/push")

type PushFlag struct {
	Flag      string
	Attribute *string
}

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Container image integration Massdriver",
}

var imagePushCmd = &cobra.Command{
	Use:   "push <namespace>/<image-name>",
	Short: "Push an image to ECR, ACR or GAR",
	Long:  imagePushCmdHelp,
	RunE:  runImagePush,
	Args:  cobra.ExactArgs(1),
}

var pushInput = image.PushImageInput{}

func init() {
	rootCmd.AddCommand(imageCmd)
	imageCmd.AddCommand(imagePushCmd)
	imagePushCmd.Flags().StringVarP(&pushInput.DockerBuildContext, "build-context", "b", ".", "Path to the directory to build the image from")
	imagePushCmd.Flags().StringVarP(&pushInput.Dockerfile, "dockerfile", "f", "Dockerfile", "Name of the dockerfile to build from if you have named it anything other than Dockerfile")
	imagePushCmd.Flags().StringSliceVarP(&pushInput.Tags, "image-tag", "t", []string{"latest"}, "Unique identifier for this version of the image")
	imagePushCmd.Flags().StringVarP(&pushInput.ArtifactID, "artifact", "a", "", "Massdriver ID of the artifact used to create the repository and generate repository credentials")
	_ = imagePushCmd.MarkFlagRequired("artifact")
	imagePushCmd.Flags().StringVarP(&pushInput.Location, "region", "r", "", "Cloud region to push the image to")
	_ = imagePushCmd.MarkFlagRequired("region")
	imagePushCmd.Flags().StringVarP(&pushInput.TargetPlatform, "platform", "p", "linux/amd64", "Set platform if server is multi-platform capable")
	imagePushCmd.Flags().StringVarP(&pushInput.CacheFrom, "cache-from", "c", "", "Path to image used for caching")
}

func runImagePush(cmd *cobra.Command, args []string) error {
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
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

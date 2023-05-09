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

var dockerBuildContext string
var dockerfileName string
var targetPlatform string
var tag string
var artifactID string
var region string

func init() {
	rootCmd.AddCommand(imageCmd)
	imageCmd.AddCommand(imagePushCmd)
	imagePushCmd.Flags().StringVarP(&dockerBuildContext, "build-context", "b", ".", "Path to the directory to build the image from")
	imagePushCmd.Flags().StringVarP(&dockerfileName, "dockerfile", "f", "Dockerfile", "Name of the dockerfile to build from if you have named it anything other than Dockerfile")
	imagePushCmd.Flags().StringVarP(&tag, "image-tag", "t", "latest", "Unique identifier for this version of the image")
	imagePushCmd.Flags().StringVarP(&artifactID, "artifact", "a", "", "Massdriver ID of the artifact used to create the repository and generate repository credentials.")
	_ = imagePushCmd.MarkFlagRequired("artifact")
	imagePushCmd.Flags().StringVarP(&region, "region", "r", "", "Cloud region to push the image to")
	_ = imagePushCmd.MarkFlagRequired("region")
	imagePushCmd.Flags().StringVarP(&targetPlatform, "platform", "p", "linux/amd64", "")
}

func runImagePush(cmd *cobra.Command, args []string) error {
	config, configErr := config.Get()
	if configErr != nil {
		return configErr
	}
	pushInput := image.PushImageInput{
		OrganizationID: config.OrgID,
		ImageName:      args[0],
	}

	err := validatePushInputAndAddFlags(&pushInput, cmd)

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

func validatePushInputAndAddFlags(input *image.PushImageInput, cmd *cobra.Command) error {
	flagsToSet := []PushFlag{
		{Flag: "dockerfile", Attribute: &input.Dockerfile},
		{Flag: "build-context", Attribute: &input.DockerBuildContext},
		{Flag: "platform", Attribute: &input.TargetPlatform},
		{Flag: "image-tag", Attribute: &input.Tag},
		{Flag: "artifact", Attribute: &input.ArtifactID},
		{Flag: "region", Attribute: &input.Location},
	}

	for _, flag := range flagsToSet {
		value, err := cmd.Flags().GetString(flag.Flag)

		if err != nil {
			return err
		}

		*flag.Attribute = value
	}

	if invalidImageName(input.ImageName) {
		return fmt.Errorf("%s is an invalid image name. Massdriver enforces the practice of namespacing images. please enter an image name in the format of namespace/image-name", input.ImageName)
	}

	return nil
}

func invalidImageName(imageName string) bool {
	return !(len(strings.Split(imageName, "/")) == 2)
}

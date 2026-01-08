package cmd

import (
	"fmt"

	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/massdriver-cloud/mass/pkg/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
func NewCmdVersion() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Display CLI version information",
		Long:    ``,
		Run:     runVersion,
		Example: `  # Show version information
  mass version`,
	}
	return versionCmd
}

func runVersion(cmd *cobra.Command, args []string) {
	latestVersion, err := version.GetLatestVersion()
	if err != nil {
		fmt.Printf("Could not check for newer version, skipping. url:%s error:%s", version.LatestReleaseURL, err.Error())
		return
	}
	isOld, _ := version.CheckForNewerVersionAvailable(latestVersion)
	if isOld {
		fmt.Printf("A newer version of the CLI is available, you can download it here: %v\n", version.LatestReleaseURL)
	}
	massVersionColor := prettylogs.Green(version.MassVersion())
	fmt.Printf("Mass CLI version: %v (git SHA: %v) \n", massVersionColor, version.MassGitSHA())
}

package cmd

import (
	"fmt"

	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/massdriver-cloud/mass/pkg/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Version of Mass CLI",
	Long:    ``,
	RunE:    runVersion,
}

func runVersion(cmd *cobra.Command, args []string) error {
	isOld, _, err := version.CheckForNewerVersionAvailable()
	if err != nil {
		fmt.Printf("could not check for newer versions at %v: %v. skipping...\n", version.LatestReleaseURL, err.Error())
	} else if isOld {
		fmt.Printf("A newer version of the CLI is available, you can download it here: %v\n", version.LatestReleaseURL)
	}
	var massVersionColor = prettylogs.Green(version.MassVersion())
	fmt.Printf("Mass CLI version: %v (git SHA: %v) \n", massVersionColor, version.MassGitSHA())
	return nil
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

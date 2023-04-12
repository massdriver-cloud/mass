package cmd

import (
	"fmt"

	"github.com/massdriver-cloud/mass/internal/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Version of Mass CLI",
	Long:    ``,
	Run:     runVersion,
}

func runVersion(cmd *cobra.Command, args []string) {
	isOld, latestVersion, err := version.CheckForNewerVersionAvailable()
	if err != nil {
		fmt.Printf("could not check for newer versions: %v. skipping...\n", err.Error())
	} else if isOld {
		fmt.Printf("A newer version of the CLI is available, you can download it here: %v\n", version.LatestReleaseURL)
	}
	fmt.Printf("Mass CLI version: %v\n", latestVersion)
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

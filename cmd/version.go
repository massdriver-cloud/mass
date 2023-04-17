package cmd

import (
	"fmt"

	"github.com/massdriver-cloud/mass/internal/prettylogs"
	"github.com/massdriver-cloud/mass/internal/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Version of Mass CLI",
	Long:    ``,
	RunE:     runVersion,
}

func runVersion(cmd *cobra.Command, args []string) error {

	isOld, latestVersion, err := version.CheckForNewerVersionAvailable()
	if err != nil {
		return fmt.Errorf("could not check for newer versions: %w. skipping...\n", err)
	} else if isOld {
		fmt.Printf("A newer version of the CLI is available, you can download it here: %v\n", version.LatestReleaseURL)
	}
	var outputColor = prettylogs.Green(latestVersion)
	fmt.Printf("Mass CLI version: %v\n", outputColor)
return nil
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

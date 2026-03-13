package cmd

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/prettylogs"
	"github.com/massdriver-cloud/mass/pkg/version"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

// NewCmdVersion returns a cobra command that prints the current CLI version.
func NewCmdVersion() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Version of Mass CLI",
		Long:    ``,
		Run:     runVersion,
	}
	return versionCmd
}

func runVersion(cmd *cobra.Command, args []string) {
	massVersionColor := prettylogs.Green(version.MassVersion())
	fmt.Printf("🧰 CLI version: %v (git SHA: %v)\n", massVersionColor, version.MassGitSHA())

	// Best-effort: check whether a newer CLI is available (does not affect exit code).
	if latestVersion, err := version.GetLatestVersion(); err == nil {
		if isOld, _ := version.CheckForNewerVersionAvailable(latestVersion); isOld {
			fmt.Printf("⬆️ A newer version of the CLI is available, you can download it here: %v\n", version.LatestReleaseURL)
		}
	}

	// Best-effort: if we can authenticate, show the Massdriver server version too.
	ctx := context.Background()
	mdClient, err := client.New()
	if err != nil {
		return
	}

	if server, err := api.GetServer(ctx, mdClient); err == nil && server != nil && server.Version != "" {
		fmt.Printf("🌐 Server version: %v\n", prettylogs.Green(server.Version))
	}
}

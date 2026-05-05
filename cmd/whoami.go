package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

// NewCmdWhoami returns a cobra command that prints the currently
// authenticated viewer.
func NewCmdWhoami() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Show the currently authenticated user or service account",
		RunE:  runWhoami,
	}
	cmd.Flags().StringP("output", "o", "text", "Output format (text or json)")
	return cmd
}

func runWhoami(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cmd.SilenceUsage = true

	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	viewer, viewerErr := api.GetViewer(ctx, mdClient)
	if viewerErr != nil {
		return viewerErr
	}

	switch outputFormat {
	case "json":
		bytes, marshalErr := json.MarshalIndent(viewer, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal viewer to JSON: %w", marshalErr)
		}
		fmt.Println(string(bytes))
	case "text":
		printViewer(viewer)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}

func printViewer(v *api.Viewer) {
	switch v.Kind {
	case api.ViewerKindAccount:
		fmt.Println("👤 User")
		fmt.Printf("   ID:    %s\n", v.ID)
		fmt.Printf("   Email: %s\n", v.Email)
		if name := strings.TrimSpace(v.FirstName + " " + v.LastName); name != "" {
			fmt.Printf("   Name:  %s\n", name)
		}
	case api.ViewerKindServiceAccount:
		fmt.Println("🤖 Service account")
		fmt.Printf("   ID:   %s\n", v.ID)
		fmt.Printf("   Name: %s\n", v.Name)
		if v.Description != "" {
			fmt.Printf("   Description: %s\n", v.Description)
		}
	}
	if v.Organization != nil {
		fmt.Printf("   Organization: %s (%s)\n", v.Organization.Name, v.Organization.ID)
	}
}

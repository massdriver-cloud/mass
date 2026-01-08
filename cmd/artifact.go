package cmd

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/pkg/api"
	"github.com/massdriver-cloud/mass/pkg/artifact"
	artifactcmd "github.com/massdriver-cloud/mass/pkg/commands/artifact"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

//go:embed templates/artifact.get.md.tmpl
var artifactTemplates embed.FS

func NewCmdArtifact() *cobra.Command {
	artifactCmd := &cobra.Command{
		Use:   "artifact",
		Short: "Manage artifacts",
		Long:  helpdocs.MustRender("artifact"),
	}

	// Import
	artifactImportCmd := &cobra.Command{
		Use:     `import`,
		Short:   "Import a custom artifact into Massdriver",
		Long:    helpdocs.MustRender("artifact/import"),
		RunE:    runArtifactImport,
		Example: `  # Import an artifact
  mass artifact import --name my-artifact --type massdriver/aws-iam-role --file artifact.json`,
	}
	artifactImportCmd.Flags().StringP("name", "n", "", "Artifact name")
	artifactImportCmd.Flags().StringP("type", "t", "", "Artifact type")
	artifactImportCmd.Flags().StringP("file", "f", "", "Artifact file")
	artifactImportCmd.MarkFlagRequired("name")
	artifactImportCmd.MarkFlagRequired("type")
	artifactImportCmd.MarkFlagRequired("file")

	// Get
	artifactGetCmd := &cobra.Command{
		Use:   "get [artifact-id]",
		Short: "Get an artifact from Massdriver",
		Long:  helpdocs.MustRender("artifact/get"),
		Args:  cobra.ExactArgs(1),
		RunE:  runArtifactGet,
		Example: `  # Get artifact using UUID (imported artifacts)
  mass artifact get 12345678-1234-1234-1234-123456789012

  # Get artifact using friendly slug (provisioned artifacts)
  mass artifact get api-prod-database-connection
  mass artifact get api-prod-grpcapi-host -o json`,
	}
	artifactGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	// Download
	artifactDownloadCmd := &cobra.Command{
		Use:   "download [artifact-id]",
		Short: "Download an artifact in the specified format",
		Long:  helpdocs.MustRender("artifact/download"),
		Args:  cobra.ExactArgs(1),
		RunE:  runArtifactDownload,
		Example: `  # Download artifact using UUID (imported artifacts)
  mass artifact download 12345678-1234-1234-1234-123456789012

  # Download artifact using friendly slug (provisioned artifacts)
  mass artifact download api-prod-database-connection
  mass artifact download network-useast1-vpc-network -f yaml`,
	}
	artifactDownloadCmd.Flags().StringP("format", "f", "json", "Download format (json, yaml, etc.)")

	artifactCmd.AddCommand(artifactImportCmd)
	artifactCmd.AddCommand(artifactGetCmd)
	artifactCmd.AddCommand(artifactDownloadCmd)

	return artifactCmd
}

func runArtifactImport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	artifactName, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	artifactType, err := cmd.Flags().GetString("type")
	if err != nil {
		return err
	}
	artifactFile, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	promptData := artifact.ImportedArtifact{Name: artifactName, Type: artifactType, File: artifactFile}
	promptErr := artifact.RunArtifactImportPrompt(ctx, mdClient, &promptData)
	if promptErr != nil {
		return promptErr
	}

	_, importErr := artifactcmd.RunImport(ctx, mdClient, promptData.Name, promptData.Type, promptData.File)
	return importErr
}

func runArtifactGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	artifactID := args[0]
	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	artifact, getErr := api.GetArtifact(ctx, mdClient, artifactID)
	if getErr != nil {
		return fmt.Errorf("error getting artifact: %w", getErr)
	}

	switch outputFormat {
	case "json":
		jsonBytes, err := json.MarshalIndent(artifact, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal artifact to JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		err = renderArtifact(artifact)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}

func runArtifactDownload(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	artifactID := args[0]
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	rendered, downloadErr := api.DownloadArtifact(ctx, mdClient, artifactID, format)
	if downloadErr != nil {
		return fmt.Errorf("error downloading artifact: %w", downloadErr)
	}

	fmt.Print(rendered)
	return nil
}

func renderArtifact(artifact *api.Artifact) error {
	specsJSON := "{}"
	if artifact.Specs != nil {
		specsBytes, err := json.MarshalIndent(artifact.Specs, "", "  ")
		if err == nil {
			specsJSON = string(specsBytes)
		}
	}

	tmplBytes, err := artifactTemplates.ReadFile("templates/artifact.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("artifact").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := struct {
		ID                 string
		Name               string
		Type               string
		Field              string
		Origin             string
		SpecsJSON          string
		Formats            []string
		CreatedAt          string
		UpdatedAt          string
		ArtifactDefinition *api.ArtifactDefinitionWithSchema
		Package            *api.ArtifactPackage
	}{
		ID:                 artifact.ID,
		Name:               artifact.Name,
		Type:               artifact.Type,
		Field:              artifact.Field,
		Origin:             artifact.Origin,
		SpecsJSON:          specsJSON,
		Formats:            artifact.Formats,
		CreatedAt:          artifact.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:          artifact.UpdatedAt.Format("2006-01-02 15:04:05"),
		ArtifactDefinition: artifact.ArtifactDefinition,
		Package:            artifact.Package,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err != nil {
		return err
	}

	out, err := r.Render(buf.String())
	if err != nil {
		return err
	}

	fmt.Print(out)
	return nil
}

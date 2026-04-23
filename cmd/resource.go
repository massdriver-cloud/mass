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
	"github.com/massdriver-cloud/mass/internal/api/v1"
	"github.com/massdriver-cloud/mass/internal/commands/resource"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

//go:embed templates/resource.get.md.tmpl
var resourceTemplates embed.FS

// NewCmdResource returns a cobra command for managing resources.
func NewCmdResource() *cobra.Command {
	resourceCmd := &cobra.Command{
		Use:   "resource",
		Short: "Manage resources",
		Long:  helpdocs.MustRender("resource"),
	}

	// Create
	resourceCreateCmd := &cobra.Command{
		Use:     `create`,
		Short:   "Create a resource",
		Aliases: []string{"import"},
		Long:    helpdocs.MustRender("resource/create"),
		RunE:    runResourceCreate,
	}
	resourceCreateCmd.Flags().StringP("name", "n", "", "Resource name")
	resourceCreateCmd.Flags().StringP("type", "t", "", "Resource type")
	resourceCreateCmd.Flags().StringP("file", "f", "", "Resource file")
	_ = resourceCreateCmd.MarkFlagRequired("name")
	_ = resourceCreateCmd.MarkFlagRequired("type")
	_ = resourceCreateCmd.MarkFlagRequired("file")

	// Get
	resourceGetCmd := &cobra.Command{
		Use:   "get [resource-id]",
		Short: "Get an resource from Massdriver",
		Long:  helpdocs.MustRender("resource/get"),
		Args:  cobra.ExactArgs(1),
		RunE:  runResourceGet,
		Example: `  # Get resource using UUID (imported resources)
  mass resource get 12345678-1234-1234-1234-123456789012

  # Get resource using friendly slug (provisioned resources)
  mass resource get api-prod-database-connection
  mass resource get api-prod-grpcapi-host -o json`,
	}
	resourceGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	// Download
	resourceDownloadCmd := &cobra.Command{
		Use:   "download [resource-id]",
		Short: "Download an resource in the specified format",
		Long:  helpdocs.MustRender("resource/download"),
		Args:  cobra.ExactArgs(1),
		RunE:  runResourceDownload,
		Example: `  # Download resource using UUID (imported resources)
  mass resource download 12345678-1234-1234-1234-123456789012

  # Download resource using friendly slug (provisioned resources)
  mass resource download api-prod-database-connection
  mass resource download network-useast1-vpc-network -f yaml`,
	}
	resourceDownloadCmd.Flags().StringP("format", "f", "json", "Download format (json, yaml, etc.)")

	// Update
	resourceUpdateCmd := &cobra.Command{
		Use:   "update [resource-id]",
		Short: "Update an imported resource",
		Long:  helpdocs.MustRender("resource/update"),
		Args:  cobra.ExactArgs(1),
		RunE:  runResourceUpdate,
		Example: `  # Update resource payload
  mass resource update 12345678-1234-1234-1234-123456789012 -f resource.json

  # Update resource payload and rename
  mass resource update 12345678-1234-1234-1234-123456789012 -f resource.json -n new-name`,
	}
	resourceUpdateCmd.Flags().StringP("name", "n", "", "New resource name")
	resourceUpdateCmd.Flags().StringP("file", "f", "", "Resource payload file")
	_ = resourceUpdateCmd.MarkFlagRequired("file")

	resourceCmd.AddCommand(resourceCreateCmd)
	resourceCmd.AddCommand(resourceGetCmd)
	resourceCmd.AddCommand(resourceDownloadCmd)
	resourceCmd.AddCommand(resourceUpdateCmd)

	return resourceCmd
}

func runResourceCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	resourceName, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	resourceType, err := cmd.Flags().GetString("type")
	if err != nil {
		return err
	}
	resourceFile, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	promptData := resource.CreatePrompt{Name: resourceName, Type: resourceType, File: resourceFile}
	promptErr := resource.RunCreatePrompt(ctx, mdClient, &promptData)
	if promptErr != nil {
		return promptErr
	}

	_, createErr := resource.RunCreate(ctx, mdClient, promptData.Name, promptData.Type, promptData.File)
	return createErr
}

func runResourceUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	resourceID := args[0]
	resourceName, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	resourceFile, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	_, updateErr := resource.RunUpdate(ctx, mdClient, resourceID, resourceName, resourceFile)
	return updateErr
}

//nolint:dupl // runResourceGet and runDefinitionGet share the same output-format pattern; refactoring would add complexity for marginal gain
func runResourceGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	resourceID := args[0]
	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	resource, getErr := api.GetResource(ctx, mdClient, resourceID)
	if getErr != nil {
		return fmt.Errorf("error getting resource: %w", getErr)
	}

	switch outputFormat {
	case "json":
		jsonBytes, marshalErr := json.MarshalIndent(resource, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal resource to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		err = renderResource(resource)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}

func runResourceDownload(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	resourceID := args[0]
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	rendered, downloadErr := api.ExportResource(ctx, mdClient, resourceID, format)
	if downloadErr != nil {
		return fmt.Errorf("error downloading resource: %w", downloadErr)
	}

	fmt.Print(rendered)
	return nil
}

func renderResource(resource *api.Resource) error {
	prettyPayload := "{}"
	if resource.Payload != nil {
		payloadBytes, err := json.MarshalIndent(resource.Payload, "", "  ")
		if err == nil {
			prettyPayload = string(payloadBytes)
		}
	}

	tmplBytes, err := resourceTemplates.ReadFile("templates/resource.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("resource").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := struct {
		ID           string
		Name         string
		Type         string
		Field        string
		Origin       string
		Payload      string
		Formats      []string
		CreatedAt    string
		UpdatedAt    string
		ResourceType *api.ResourceType
		Instance     *api.Instance
	}{
		ID:           resource.ID,
		Name:         resource.Name,
		Type:         resource.ResourceType.Name,
		Field:        resource.Field,
		Origin:       resource.Origin,
		Payload:      prettyPayload,
		Formats:      resource.Formats,
		CreatedAt:    resource.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    resource.UpdatedAt.Format("2006-01-02 15:04:05"),
		ResourceType: resource.ResourceType,
		Instance:     resource.Instance,
	}

	var buf bytes.Buffer
	if renderErr := tmpl.Execute(&buf, data); renderErr != nil {
		return fmt.Errorf("failed to execute template: %w", renderErr)
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

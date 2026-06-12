package cmd

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/mass/internal/commands/resource"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/resources"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
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

	resourceCmd.AddCommand(newResourceCreateCmd())
	resourceCmd.AddCommand(newResourceGetCmd())
	resourceCmd.AddCommand(newResourceDownloadCmd())
	resourceCmd.AddCommand(newResourceUpdateCmd())
	resourceCmd.AddCommand(newResourceDeleteCmd())
	resourceCmd.AddCommand(newResourceListCmd())

	return resourceCmd
}

func newResourceCreateCmd() *cobra.Command {
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
	return resourceCreateCmd
}

func newResourceGetCmd() *cobra.Command {
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
	return resourceGetCmd
}

func newResourceDownloadCmd() *cobra.Command {
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
	return resourceDownloadCmd
}

func newResourceUpdateCmd() *cobra.Command {
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
	return resourceUpdateCmd
}

func newResourceDeleteCmd() *cobra.Command {
	resourceDeleteCmd := &cobra.Command{
		Use:   "delete [resource-id]",
		Short: "Delete a resource",
		Args:  cobra.ExactArgs(1),
		RunE:  runResourceDelete,
		Example: `  # Delete an imported resource
  mass resource delete 12345678-1234-1234-1234-123456789012

  # Skip the confirmation prompt
  mass resource delete 12345678-1234-1234-1234-123456789012 --force`,
	}
	resourceDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	return resourceDeleteCmd
}

func newResourceListCmd() *cobra.Command {
	resourceListCmd := &cobra.Command{
		Use:     "list",
		Short:   "List resources",
		Aliases: []string{"ls"},
		Long:    helpdocs.MustRender("resource/list"),
		RunE:    runResourceList,
		Example: `  # List all resources
  mass resource list

  # Search and filter
  mass resource list --search database
  mass resource list --type aws-iam-role --origin provisioned
  mass resource list --environment ecomm-prod -o json`,
	}
	resourceListCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	resourceListCmd.Flags().StringP("search", "s", "", "Full-text search across resource name")
	resourceListCmd.Flags().StringP("type", "t", "", "Filter by resource type id (e.g. aws-iam-role)")
	resourceListCmd.Flags().String("origin", "", "Filter by origin (imported, provisioned). Empty matches both")
	resourceListCmd.Flags().StringP("environment", "e", "", "Limit to provisioned resources in an environment")
	resourceListCmd.Flags().String("sort", "", "Sort field (name, created_at)")
	resourceListCmd.Flags().String("order", "asc", "Sort order (asc, desc)")
	return resourceListCmd
}

func parseResourceSortField(s string) (resources.SortField, error) {
	switch strings.ToLower(s) {
	case "", "name":
		return resources.SortByName, nil
	case "created_at":
		return resources.SortByCreatedAt, nil
	default:
		return "", fmt.Errorf("unknown sort field %q (valid: name, created_at)", s)
	}
}

func parseSortOrder(s string) (resources.SortOrder, error) {
	switch strings.ToLower(s) {
	case "", "asc":
		return resources.SortAsc, nil
	case "desc":
		return resources.SortDesc, nil
	default:
		return "", fmt.Errorf("unknown sort order %q (valid: asc, desc)", s)
	}
}

func runResourceList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	search, _ := cmd.Flags().GetString("search")
	resourceType, _ := cmd.Flags().GetString("type")
	origin, _ := cmd.Flags().GetString("origin")
	environmentID, _ := cmd.Flags().GetString("environment")
	sortField, _ := cmd.Flags().GetString("sort")
	sortOrder, _ := cmd.Flags().GetString("order")

	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	listInput := resources.ListInput{
		Search:        search,
		ResourceType:  resourceType,
		EnvironmentID: environmentID,
	}
	if origin != "" {
		listInput.Origin = resources.Origin(strings.ToUpper(origin))
	}
	if sortField != "" || cmd.Flags().Changed("order") {
		sortBy, sortByErr := parseResourceSortField(sortField)
		if sortByErr != nil {
			return sortByErr
		}
		order, orderErr := parseSortOrder(sortOrder)
		if orderErr != nil {
			return orderErr
		}
		listInput.SortBy = sortBy
		listInput.SortOrder = order
	}

	seq := mdClient.Resources.Iter(ctx, listInput)

	switch output {
	case "json":
		res, collectErr := types.Collect(seq)
		if collectErr != nil {
			return collectErr
		}
		jsonBytes, marshalErr := json.MarshalIndent(res, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal resources to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
		return nil
	case "table":
		return cli.Paginate(seq, cli.PagerConfig[resources.Resource]{
			Columns: []string{"ID", "Name", "Type", "Origin", "Created At"},
			Row: func(r resources.Resource) []string {
				typeName := ""
				if r.ResourceType != nil {
					typeName = r.ResourceType.Name
				}
				return []string{r.ID, r.Name, typeName, strings.ToLower(r.Origin), r.CreatedAt.Format("2006-01-02 15:04:05")}
			},
		})
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
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

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	api := resource.NewAPI(mdClient)
	promptData := resource.CreatePrompt{Name: resourceName, Type: resourceType, File: resourceFile}
	if err := resource.RunCreatePrompt(ctx, api, &promptData); err != nil {
		return err
	}

	_, createErr := resource.RunCreate(ctx, api, promptData.Name, promptData.Type, promptData.File)
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

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	_, updateErr := resource.RunUpdate(ctx, resource.NewAPI(mdClient), resourceID, resourceName, resourceFile)
	return updateErr
}

func runResourceDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	resourceID := args[0]
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	return resource.RunDelete(ctx, resource.NewAPI(mdClient), resourceID, force, os.Stdin)
}

func runResourceGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	resourceID := args[0]
	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	res, err := mdClient.Resources.Get(ctx, resourceID)
	if err != nil {
		return fmt.Errorf("error getting resource: %w", err)
	}

	switch outputFormat {
	case "json":
		jsonBytes, marshalErr := json.MarshalIndent(res, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal resource to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		err = renderResource(res)
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

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	exported, err := mdClient.Resources.Export(ctx, resourceID, format)
	if err != nil {
		return fmt.Errorf("error downloading resource: %w", err)
	}

	fmt.Print(exported.Rendered)
	return nil
}

func renderResource(res *types.Resource) error {
	prettyPayload := "{}"
	if res.Payload != nil {
		payloadBytes, err := json.MarshalIndent(res.Payload, "", "  ")
		if err == nil {
			prettyPayload = string(payloadBytes)
		}
	}

	tmplBytes, err := resourceTemplates.ReadFile("templates/resource.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("resource").Funcs(cli.MarkdownTemplateFuncs).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	typeName := ""
	if res.ResourceType != nil {
		typeName = res.ResourceType.Name
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
		ResourceType *types.ResourceType
		Instance     *types.Instance
	}{
		ID:           res.ID,
		Name:         res.Name,
		Type:         typeName,
		Field:        res.Field,
		Origin:       res.Origin,
		Payload:      prettyPayload,
		Formats:      res.Formats,
		CreatedAt:    res.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    res.UpdatedAt.Format("2006-01-02 15:04:05"),
		ResourceType: res.ResourceType,
		Instance:     res.Instance,
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

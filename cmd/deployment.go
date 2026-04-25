package cmd

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/cli"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

//go:embed templates/deployment.get.md.tmpl
var deploymentTemplates embed.FS

// NewCmdDeployment returns a cobra command for managing deployments.
func NewCmdDeployment() *cobra.Command {
	deploymentCmd := &cobra.Command{
		Use:     "deployment",
		Aliases: []string{"dep"},
		Short:   "Manage deployments",
		Long:    helpdocs.MustRender("deployment"),
	}

	deploymentGetCmd := &cobra.Command{
		Use:     "get <deployment-id>",
		Short:   "Get a deployment by ID",
		Example: `mass deployment get 12345678-1234-1234-1234-123456789012`,
		Long:    helpdocs.MustRender("deployment/get"),
		Args:    cobra.ExactArgs(1),
		RunE:    runDeploymentGet,
	}
	deploymentGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	deploymentListCmd := &cobra.Command{
		Use:     "list <instance-id>",
		Aliases: []string{"ls"},
		Short:   "List deployments for an instance (most recent first)",
		Example: `mass deployment list ecomm-prod-db --limit 25`,
		Long:    helpdocs.MustRender("deployment/list"),
		Args:    cobra.ExactArgs(1),
		RunE:    runDeploymentList,
	}
	deploymentListCmd.Flags().IntP("limit", "n", 10, "Maximum number of deployments to return (max 100)")

	deploymentLogsCmd := &cobra.Command{
		Use:     "logs <deployment-id>",
		Short:   "Stream the log output from a deployment",
		Example: `mass deployment logs 12345678-1234-1234-1234-123456789012`,
		Long:    helpdocs.MustRender("deployment/logs"),
		Args:    cobra.ExactArgs(1),
		RunE:    runDeploymentLogs,
	}

	deploymentCmd.AddCommand(deploymentGetCmd)
	deploymentCmd.AddCommand(deploymentListCmd)
	deploymentCmd.AddCommand(deploymentLogsCmd)

	return deploymentCmd
}

func runDeploymentGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	deploymentID := args[0]

	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	deployment, err := api.GetDeployment(ctx, mdClient, deploymentID)
	if err != nil {
		return err
	}

	switch outputFormat {
	case "json":
		jsonBytes, marshalErr := json.MarshalIndent(deployment, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal deployment to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		return renderDeployment(deployment)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}

func runDeploymentList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	instanceID := args[0]

	limit, err := cmd.Flags().GetInt("limit")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	filter := &api.DeploymentsFilter{
		InstanceId: &api.IdFilter{Eq: instanceID},
	}
	sort := &api.DeploymentsSort{
		Field: api.DeploymentsSortFieldCreatedAt,
		Order: api.SortOrderDesc,
	}
	deployments, err := api.ListDeployments(ctx, mdClient, filter, sort, limit)
	if err != nil {
		return err
	}

	tbl := cli.NewTable("ID", "Action", "Status", "Version", "Created At", "By", "Message")
	for _, d := range deployments {
		tbl.AddRow(d.ID, d.Action, d.Status, d.Version, d.CreatedAt, d.DeployedBy, cli.TruncateString(d.Message, 40))
	}
	tbl.Print()

	return nil
}

func runDeploymentLogs(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	deploymentID := args[0]
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	logs, err := api.GetDeploymentLogs(ctx, mdClient, deploymentID)
	if err != nil {
		return fmt.Errorf("error getting deployment logs: %w", err)
	}

	for _, log := range logs {
		fmt.Fprint(os.Stdout, log.Message)
	}

	return nil
}

func renderDeployment(deployment *api.Deployment) error {
	tmplBytes, err := deploymentTemplates.ReadFile("templates/deployment.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("deployment").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if renderErr := tmpl.Execute(&buf, deployment); renderErr != nil {
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

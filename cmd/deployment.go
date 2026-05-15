package cmd

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/template"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/deployments"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
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

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	deployment, err := mdClient.Deployments.Get(ctx, deploymentID)
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

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	// SDK's List auto-paginates without a total cap; we want a total cap, so
	// drive Iter and stop after `limit` items.
	listInput := deployments.ListInput{
		InstanceID: instanceID,
		SortBy:     deployments.SortByCreatedAt,
		SortOrder:  deployments.SortDesc,
	}

	tbl := cli.NewTable("ID", "Action", "Status", "Version", "Created At", "By", "Message")
	count := 0
	for d, iterErr := range mdClient.Deployments.Iter(ctx, listInput) {
		if iterErr != nil {
			return iterErr
		}
		if limit > 0 && count >= limit {
			break
		}
		tbl.AddRow(d.ID, d.Action, d.Status, d.Version, d.CreatedAt, d.DeployedBy, cli.TruncateString(d.Message, 40))
		count++
	}
	tbl.Print()

	return nil
}

func runDeploymentLogs(cmd *cobra.Command, args []string) error {
	deploymentID := args[0]
	cmd.SilenceUsage = true

	ctx, cancel := signalContext(context.Background())
	defer cancel()

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	// TailLogs collapses backfill + terminal-check + live streaming into one call.
	tailErr := mdClient.Deployments.TailLogs(ctx, deploymentID, os.Stdout)
	if errors.Is(tailErr, deployments.ErrStreamingRequiresPAT) {
		// Fall back to a one-shot static-log dump so the user still gets the
		// available history. Streaming would require a personal access token.
		fmt.Fprintln(os.Stderr, "warning: log streaming requires a personal access token (mds_*/md_*); showing static logs instead")
		backfill, err := mdClient.Deployments.GetLogs(ctx, deploymentID)
		if err != nil {
			return fmt.Errorf("error getting deployment logs: %w", err)
		}
		fmt.Fprint(os.Stdout, backfill)
		return nil
	}
	if tailErr != nil && !errors.Is(tailErr, context.Canceled) {
		return tailErr
	}
	return nil
}

// signalContext returns a derived context that cancels on SIGINT/SIGTERM, so
// Ctrl-C cleanly tears down the WebSocket and exits.
func signalContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
}

//nolint:dupl // parallel template-render shape with renderInstance; consolidating would couple unrelated commands
func renderDeployment(deployment *types.Deployment) error {
	tmplBytes, err := deploymentTemplates.ReadFile("templates/deployment.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("deployment").Funcs(cli.MarkdownTemplateFuncs).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	paramsJSON := "{}"
	if deployment.Params != nil {
		if b, marshalErr := json.MarshalIndent(deployment.Params, "", "  "); marshalErr == nil {
			paramsJSON = string(b)
		}
	}

	data := struct {
		*types.Deployment
		ParamsJSON string
	}{Deployment: deployment, ParamsJSON: paramsJSON}

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

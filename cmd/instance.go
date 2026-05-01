package cmd

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/mass/internal/commands/instance"
	"github.com/massdriver-cloud/mass/internal/files"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

//go:embed templates/instance.get.md.tmpl
var instanceTemplates embed.FS

// NewCmdInstance returns a cobra command for managing instances of IaC deployed in environments.
func NewCmdInstance() *cobra.Command { //nolint:funlen // cobra command builders are necessarily long
	instanceCmd := &cobra.Command{
		Use:     "instance",
		Aliases: []string{"inst", "package", "instance"},
		Short:   "Manage instances of IaC deployed in environments.",
		Long:    helpdocs.MustRender("instance"),
	}

	instanceDeployCmd := &cobra.Command{
		Use:     `deploy <project>-<env>-<manifest>`,
		Short:   "Deploy instances",
		Example: `mass instance deploy ecomm-prod-vpc`,
		Long:    helpdocs.MustRender("instance/deploy"),
		Args:    cobra.ExactArgs(1),
		RunE:    runInstanceDeploy,
	}
	instanceDeployCmd.Flags().StringP("message", "m", "", "Add a message when deploying")
	instanceDeployCmd.Flags().StringP("params", "p", "", "Path to params json, tfvars or yaml file. Use '-' to read from stdin. When provided, the full configuration is replaced. Supports bash interpolation.")
	instanceDeployCmd.Flags().StringArrayP("patch", "P", []string{}, "Patch the last deployed configuration using a JQ expression. Can be specified multiple times.")
	instanceDeployCmd.Flags().BoolP("follow", "f", false, "Stream the deployment's logs to stdout until it completes")
	instanceDeployCmd.MarkFlagsMutuallyExclusive("params", "patch")

	instanceExportCmd := &cobra.Command{
		Use:     `export <project>-<env>-<manifest>`,
		Short:   "Export instances",
		Example: `mass instance export ecomm-prod-vpc`,
		Long:    helpdocs.MustRender("instance/export"),
		Args:    cobra.ExactArgs(1),
		RunE:    runInstanceExport,
	}

	// instance and infra are the same, lets reuse a get command/template here.
	instanceGetCmd := &cobra.Command{
		Use:     `get  <project>-<env>-<manifest>`,
		Short:   "Get an instance",
		Aliases: []string{"g"},
		Example: `mass instance get ecomm-prod-vpc`,
		Long:    helpdocs.MustRender("instance/get"),
		Args:    cobra.ExactArgs(1),
		RunE:    runInstanceGet,
	}
	instanceGetCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	instanceVersionCmd := &cobra.Command{
		Use:     `version <instance-id>@<version>`,
		Short:   "Set instance version",
		Example: `mass instance version api-prod-db@latest --release-channel development`,
		Long:    helpdocs.MustRender("instance/version"),
		Args:    cobra.ExactArgs(1),
		RunE:    runInstanceVersion,
	}
	instanceVersionCmd.Flags().String("release-channel", "stable", "Release strategy (stable or development)")

	instanceDestroyCmd := &cobra.Command{
		Use:     `destroy <project>-<env>-<manifest>`,
		Short:   "Destroy (decommission) an instance",
		Example: `mass instance destroy api-prod-db --force`,
		Long:    "Destroy (decommission) an instance. This will permanently delete the instance and all its resources.",
		Args:    cobra.ExactArgs(1),
		RunE:    runInstanceDeploy,
	}
	instanceDestroyCmd.Flags().StringP("message", "m", "", "Add a message when decommissioning")
	instanceDestroyCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	instanceDestroyCmd.Flags().StringP("params", "p", "", "Path to params json, tfvars or yaml file. Use '-' to read from stdin. When provided, the full configuration is replaced. Supports bash interpolation.")
	instanceDestroyCmd.Flags().StringArrayP("patch", "P", []string{}, "Patch the last deployed configuration using a JQ expression. Can be specified multiple times.")
	instanceDestroyCmd.Flags().Bool("follow", false, "Stream the deployment's logs to stdout until it completes")
	instanceDestroyCmd.MarkFlagsMutuallyExclusive("params", "patch")

	instanceListCmd := &cobra.Command{
		Use:     `list <project>-<env>`,
		Short:   "List instances in an environment",
		Aliases: []string{"ls"},
		Example: `mass instance list ecomm-prod`,
		Long:    helpdocs.MustRender("instance/list"),
		Args:    cobra.ExactArgs(1),
		RunE:    runInstanceList,
	}

	instanceCmd.AddCommand(instanceDeployCmd)
	instanceCmd.AddCommand(instanceExportCmd)
	instanceCmd.AddCommand(instanceGetCmd)
	instanceCmd.AddCommand(instanceListCmd)
	instanceCmd.AddCommand(instanceVersionCmd)
	instanceCmd.AddCommand(instanceDestroyCmd)

	return instanceCmd
}

func runInstanceGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	instanceID := args[0]

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	instance, err := api.GetInstance(ctx, mdClient, instanceID)
	if err != nil {
		return err
	}

	switch outputFormat {
	case "json":
		jsonBytes, marshalErr := json.MarshalIndent(instance, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal instance to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		err = renderInstance(instance)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}

func renderInstance(instance *api.Instance) error {
	tmplBytes, err := instanceTemplates.ReadFile("templates/instance.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	funcMap := template.FuncMap{
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
	}
	tmpl, err := template.New("instance").Funcs(funcMap).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if renderErr := tmpl.Execute(&buf, instance); renderErr != nil {
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

func runInstanceDeploy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	name := args[0]

	action := api.DeploymentActionProvision
	if cmd.Name() == "destroy" {
		action = api.DeploymentActionDecommission
	}

	msg, err := cmd.Flags().GetString("message")
	if err != nil {
		return err
	}

	paramsPath, err := cmd.Flags().GetString("params")
	if err != nil {
		return err
	}

	patchQueries, err := cmd.Flags().GetStringArray("patch")
	if err != nil {
		return err
	}

	follow, err := cmd.Flags().GetBool("follow")
	if err != nil {
		return err
	}

	opts := instance.DeployOptions{
		Action:       action,
		Message:      msg,
		PatchQueries: patchQueries,
	}
	if follow {
		opts.LogWriter = os.Stdout
	}

	if paramsPath != "" {
		params, readErr := readParams(paramsPath)
		if readErr != nil {
			return readErr
		}
		opts.Params = params
	}

	if action == api.DeploymentActionDecommission {
		force, forceErr := cmd.Flags().GetBool("force")
		if forceErr != nil {
			return forceErr
		}
		if !force {
			fmt.Printf("WARNING: This will permanently decommission instance `%s` and all its resources.\n", name)
			fmt.Printf("Type `%s` to confirm decommission: ", name)
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			if strings.TrimSpace(answer) != name {
				fmt.Println("Decommission cancelled.")
				return nil
			}
		}
		cmd.SilenceUsage = true
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	if _, err = instance.RunDeploy(ctx, mdClient, name, opts); err != nil {
		return err
	}

	if action == api.DeploymentActionDecommission {
		fmt.Printf("✅ Instance `%s` decommission started\n", name)
		urlHelper, urlErr := api.NewURLHelper(ctx, mdClient)
		if urlErr == nil {
			fmt.Printf("🔗 %s\n", urlHelper.InstanceURL(name))
		}
	}

	return nil
}

func readParams(path string) (map[string]any, error) {
	params := map[string]any{}
	if path == "-" {
		if err := json.NewDecoder(os.Stdin).Decode(&params); err != nil {
			return nil, fmt.Errorf("failed to decode JSON from stdin: %w", err)
		}
		return params, nil
	}
	if err := files.Read(path, &params); err != nil {
		return nil, err
	}
	return params, nil
}

func runInstanceExport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	instanceID := args[0]

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	exportErr := instance.RunExport(ctx, mdClient, instanceID)
	if exportErr != nil {
		return fmt.Errorf("failed to export instance: %w", exportErr)
	}

	return nil
}

func runInstanceVersion(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	instanceIDAndVersion := args[0]

	// Parse instance-id@version format
	parts := strings.Split(instanceIDAndVersion, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format: expected <instance-id>@<version>, got %s", instanceIDAndVersion)
	}
	instanceID := parts[0]
	version := parts[1]

	releaseChannel, err := cmd.Flags().GetString("release-channel")
	if err != nil {
		return err
	}

	// Convert release channel to ReleaseStrategy enum value
	var releaseStrategy api.ReleaseStrategy
	switch releaseChannel {
	case "development":
		releaseStrategy = api.ReleaseStrategyDevelopment
	case "stable":
		releaseStrategy = api.ReleaseStrategyStable
	default:
		return fmt.Errorf("invalid release-channel: must be 'stable' or 'development', got '%s'", releaseChannel)
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	input := api.UpdateInstanceInput{
		Version:         version,
		ReleaseStrategy: releaseStrategy,
	}

	updatedInstance, err := api.UpdateInstance(ctx, mdClient, instanceID, input)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Instance `%s` version set successfully\n", updatedInstance.ID)
	urlHelper, urlErr := api.NewURLHelper(ctx, mdClient)
	if urlErr == nil {
		fmt.Printf("🔗 %s\n", urlHelper.InstanceURL(updatedInstance.ID))
	}

	return nil
}

func runInstanceList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	environmentID := args[0]

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	filter := api.InstancesFilter{
		EnvironmentId: &api.IdFilter{Eq: environmentID},
	}

	instances, err := api.ListInstances(ctx, mdClient, &filter)
	if err != nil {
		return err
	}

	tbl := cli.NewTable("ID", "Name", "Bundle", "Status")

	for _, instance := range instances {
		tbl.AddRow(instance.ID, instance.Component.Name, instance.Bundle.Name, instance.Status)
	}

	tbl.Print()

	return nil
}

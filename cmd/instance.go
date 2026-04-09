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
	apiv0 "github.com/massdriver-cloud/mass/internal/api/v0"
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/mass/internal/commands/instance"
	"github.com/massdriver-cloud/mass/internal/files"
	"github.com/massdriver-cloud/mass/internal/prettylogs"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

var (
	instanceParamsPath   = "./params.json"
	instancePatchQueries []string
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

	instanceConfigureCmd := &cobra.Command{
		Use:     `configure <project>-<env>-<manifest>`,
		Short:   "Configure instance",
		Aliases: []string{"cfg"},
		Example: `mass instance configure ecomm-prod-vpc --params=params.json`,
		Long:    helpdocs.MustRender("instance/configure"),
		Args:    cobra.ExactArgs(1),
		RunE:    runInstanceConfigure,
	}
	instanceConfigureCmd.Flags().StringVarP(&instanceParamsPath, "params", "p", instanceParamsPath, "Path to params json, tfvars or yaml file. Use '-' to read from stdin. This file supports bash interpolation.")

	instanceDeployCmd := &cobra.Command{
		Use:     `deploy <project>-<env>-<manifest>`,
		Short:   "Deploy instances",
		Example: `mass instance deploy ecomm-prod-vpc`,
		Long:    helpdocs.MustRender("instance/deploy"),
		Args:    cobra.ExactArgs(1),
		RunE:    runInstanceDeploy,
	}
	instanceDeployCmd.Flags().StringP("message", "m", "", "Add a message when deploying")

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

	instancePatchCmd := &cobra.Command{
		Use:     `patch <project>-<env>-<manifest>`,
		Short:   "Patch individual instance parameter values",
		Example: `mass instance patch ecomm-prod-db --set='.version = "13.4"'`,
		Long:    helpdocs.MustRender("instance/patch"),
		Args:    cobra.ExactArgs(1),
		RunE:    runInstancePatch,
	}
	instancePatchCmd.Flags().StringArrayVarP(&instancePatchQueries, "set", "s", []string{}, "Sets an instance parameter value using JQ expressions.")

	instanceCreateCmd := &cobra.Command{
		Use:     `create [slug]`,
		Short:   "Create a manifest (add bundle to project)",
		Example: `mass instance create dbbundle-test-serverless --bundle aws-rds-cluster`,
		Long:    helpdocs.MustRender("instance/create"),
		Args:    cobra.ExactArgs(1),
		RunE:    runPkgCreate,
	}
	instanceCreateCmd.Flags().StringP("name", "n", "", "Manifest name (defaults to slug if not provided)")
	instanceCreateCmd.Flags().StringP("bundle", "b", "", "Bundle ID or name (required)")
	_ = instanceCreateCmd.MarkFlagRequired("bundle")

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
		RunE:    runInstanceDestroy,
	}
	instanceDestroyCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	instanceResetCmd := &cobra.Command{
		Use:     `reset <project>-<env>-<manifest>`,
		Short:   "Reset instance status to 'Initialized'",
		Example: `mass instance reset api-prod-db`,
		Long:    helpdocs.MustRender("instance/reset"),
		Args:    cobra.ExactArgs(1),
		RunE:    runInstanceReset,
	}
	instanceResetCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	instanceListCmd := &cobra.Command{
		Use:     `list <project>-<env>`,
		Short:   "List instances in an environment",
		Aliases: []string{"ls"},
		Example: `mass instance list ecomm-prod`,
		Long:    helpdocs.MustRender("instance/list"),
		Args:    cobra.ExactArgs(1),
		RunE:    runInstanceList,
	}

	instanceCmd.AddCommand(instanceConfigureCmd)
	instanceCmd.AddCommand(instanceDeployCmd)
	instanceCmd.AddCommand(instanceExportCmd)
	instanceCmd.AddCommand(instanceGetCmd)
	instanceCmd.AddCommand(instanceListCmd)
	instanceCmd.AddCommand(instancePatchCmd)
	instanceCmd.AddCommand(instanceCreateCmd)
	instanceCmd.AddCommand(instanceVersionCmd)
	instanceCmd.AddCommand(instanceDestroyCmd)
	instanceCmd.AddCommand(instanceResetCmd)

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

	instance, err := apiv0.GetPackage(ctx, mdClient, instanceID)
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

func renderInstance(instance *apiv0.Package) error {
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

	msg, err := cmd.Flags().GetString("message")
	if err != nil {
		return err
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	_, err = instance.RunDeploy(ctx, mdClient, name, msg)

	return err
}

func runInstanceConfigure(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	instanceID := args[0]

	params := map[string]any{}
	if instanceParamsPath == "-" {
		// Read from stdin
		if err := json.NewDecoder(os.Stdin).Decode(&params); err != nil {
			return fmt.Errorf("failed to decode JSON from stdin: %w", err)
		}
	} else {
		if err := files.Read(instanceParamsPath, &params); err != nil {
			return err
		}
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	configuredInstance, err := instance.RunConfigure(ctx, mdClient, instanceID, params)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Instance `%s` configured successfully\n", configuredInstance.ID)

	// Get instance details to build URL
	instanceDetails, err := apiv0.GetPackage(ctx, mdClient, configuredInstance.ID)
	if err == nil && instanceDetails.Environment != nil && instanceDetails.Environment.Project != nil && instanceDetails.Manifest != nil {
		urlHelper, urlErr := apiv0.NewURLHelper(ctx, mdClient)
		if urlErr == nil {
			fmt.Printf("🔗 %s\n", urlHelper.InstanceURL(instanceDetails.Environment.Project.ID, instanceDetails.Environment.ID, instanceDetails.Manifest.ID))
		}
	}

	return nil
}

func runInstancePatch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	instanceID := args[0]

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	_, err := instance.RunPatch(ctx, mdClient, instanceID, instancePatchQueries)

	var name = lipgloss.NewStyle().SetString(instanceID).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("Patching: %s", name)
	fmt.Println(msg)

	return err
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

func runPkgCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	fullID := args[0]
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}

	bundleIDOrName, err := cmd.Flags().GetString("bundle")
	if err != nil {
		return err
	}

	// Parse project-env-manifest format: extract project (first), env (second), and manifest (third)
	// Format is $proj-$env-$manifest where each part has no hyphens
	parts := strings.Split(fullID, "-")
	if len(parts) != 3 {
		return fmt.Errorf("unable to determine project, environment, and manifest from slug %s (expected format: project-env-manifest)", fullID)
	}
	projectIDOrID := parts[0]
	environmentID := parts[1]
	manifestID := parts[2]

	if name == "" {
		name = manifestID
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	_, err = apiv0.CreateManifest(ctx, mdClient, bundleIDOrName, projectIDOrID, name, manifestID, "")
	if err != nil {
		return err
	}

	fmt.Printf("✅ Instance `%s` created successfully\n", fullID)
	urlHelper, urlErr := apiv0.NewURLHelper(ctx, mdClient)
	if urlErr == nil {
		fmt.Printf("🔗 %s\n", urlHelper.InstanceURL(projectIDOrID, environmentID, manifestID))
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
	var releaseStrategy apiv0.ReleaseStrategy
	switch releaseChannel {
	case "development":
		releaseStrategy = apiv0.ReleaseStrategyDevelopment
	case "stable":
		releaseStrategy = apiv0.ReleaseStrategyStable
	default:
		return fmt.Errorf("invalid release-channel: must be 'stable' or 'development', got '%s'", releaseChannel)
	}

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	updatedPkg, err := apiv0.SetPackageVersion(ctx, mdClient, instanceID, version, releaseStrategy)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Instance `%s` version set successfully\n", updatedPkg.ID)

	// Get instance details to build URL
	instanceDetails, err := apiv0.GetPackage(ctx, mdClient, updatedPkg.ID)
	if err == nil && instanceDetails.Environment != nil && instanceDetails.Environment.Project != nil && instanceDetails.Manifest != nil {
		urlHelper, urlErr := apiv0.NewURLHelper(ctx, mdClient)
		if urlErr == nil {
			fmt.Printf("🔗 %s\n", urlHelper.InstanceURL(instanceDetails.Environment.Project.ID, instanceDetails.Environment.ID, instanceDetails.Manifest.ID))
		}
	}

	return nil
}

func runInstanceDestroy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	instanceID := args[0]
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	// Get instance details for confirmation and URL
	instance, err := apiv0.GetPackage(ctx, mdClient, instanceID)
	if err != nil {
		return err
	}

	// Prompt for confirmation - requires typing the instance slug unless --force is used
	if !force {
		fmt.Printf("WARNING: This will permanently decommission instance `%s` and all its resources.\n", instance.ID)
		fmt.Printf("Type `%s` to confirm decommission: ", instance.ID)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		if answer != instance.ID {
			fmt.Println("Decommission cancelled.")
			return nil
		}
	}

	_, err = apiv0.DecommissionPackage(ctx, mdClient, instance.ID, "")
	if err != nil {
		return err
	}

	fmt.Printf("✅ Instance `%s` decommission started\n", instance.ID)

	// Get instance details to build URL
	if instance.Environment != nil && instance.Environment.Project != nil && instance.Manifest != nil {
		urlHelper, urlErr := apiv0.NewURLHelper(ctx, mdClient)
		if urlErr == nil {
			fmt.Printf("🔗 %s\n", urlHelper.InstanceURL(instance.Environment.Project.ID, instance.Environment.ID, instance.Manifest.ID))
		}
	}

	return nil
}

func runInstanceReset(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	instanceID := args[0]

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	// Get instance details for confirmation
	instanceDetails, err := apiv0.GetPackage(ctx, mdClient, instanceID)
	if err != nil {
		return err
	}

	// Prompt for confirmation unless --force is used
	if !force {
		fmt.Printf("%s: This will reset instance `%s` to 'Initialized' state and delete deployment history.\n", prettylogs.Orange("WARNING"), instanceDetails.ID)
		fmt.Printf("Type `%s` to confirm reset: ", instanceDetails.ID)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)

		if answer != instanceDetails.ID {
			fmt.Println("Reset cancelled.")
			return nil
		}
	}

	instance, err := instance.RunReset(ctx, mdClient, instanceID)
	if err != nil {
		return err
	}

	var name = lipgloss.NewStyle().SetString(instance.ID).Foreground(lipgloss.Color("#7D56F4"))
	msg := fmt.Sprintf("✅ Instance %s reset successfully", name)
	fmt.Println(msg)

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

	env, err := apiv0.GetEnvironment(ctx, mdClient, environmentID)
	if err != nil {
		return err
	}

	tbl := cli.NewTable("ID", "Name", "Bundle", "Status")

	for _, p := range env.Packages {
		name := ""
		if p.Manifest != nil {
			name = p.Manifest.Name
		}
		bundleName := ""
		if p.Bundle != nil {
			bundleName = p.Bundle.Name
		}
		tbl.AddRow(p.ID, name, bundleName, p.Status)
	}

	tbl.Print()

	return nil
}

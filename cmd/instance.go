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
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/mass/internal/commands/instance"
	"github.com/massdriver-cloud/mass/internal/files"
	"github.com/massdriver-cloud/mass/internal/prettylogs"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"

	"github.com/charmbracelet/glamour"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/deployments"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/instances"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
	"github.com/spf13/cobra"
)

//go:embed templates/instance.get.md.tmpl
var instanceTemplates embed.FS

// NewCmdInstance returns a cobra command for managing instances of IaC deployed in environments.
func NewCmdInstance() *cobra.Command {
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
		Example: `mass instance version api-prod-db@latest`,
		Long:    helpdocs.MustRender("instance/version"),
		Args:    cobra.ExactArgs(1),
		RunE:    runInstanceVersion,
	}

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

	instanceOrphanCmd := &cobra.Command{
		Use:     `orphan <project>-<env>-<manifest>`,
		Short:   "Orphan an instance (reset to INITIALIZED, optionally clearing state locks)",
		Example: `mass instance orphan api-prod-db --force`,
		Long:    helpdocs.MustRender("instance/orphan"),
		Args:    cobra.ExactArgs(1),
		RunE:    runInstanceOrphan,
	}
	instanceOrphanCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	instanceOrphanCmd.Flags().Bool("delete-state", false, "Also delete the remote Terraform/OpenTofu state files (irreversible)")

	instanceCopyCmd := &cobra.Command{
		Use:     `copy [source] [destination]`,
		Aliases: []string{"promote"},
		Short:   "Copy an instance's configuration into another instance of the same component",
		Example: `mass instance copy ecomm-staging-db ecomm-production-db --copy-secrets`,
		Long:    helpdocs.MustRender("instance/copy"),
		Args:    cobra.ExactArgs(2),
		RunE:    runInstanceCopy,
	}
	instanceCopyCmd.Flags().StringP("message", "m", "", "Optional message attached to the plan deployment created on the destination")
	instanceCopyCmd.Flags().StringP("overrides", "o", "", "Path to a JSON or YAML file of param overrides deep-merged onto the source params")
	instanceCopyCmd.Flags().Bool("copy-secrets", false, "Copy secrets from the source instance to the destination")
	instanceCopyCmd.Flags().Bool("copy-remote-references", false, "Copy remote-reference overrides from the source instance to the destination")

	instanceCmd.AddCommand(instanceDeployCmd)
	instanceCmd.AddCommand(instanceExportCmd)
	instanceCmd.AddCommand(instanceGetCmd)
	instanceCmd.AddCommand(instanceListCmd)
	instanceCmd.AddCommand(instanceVersionCmd)
	instanceCmd.AddCommand(instanceDestroyCmd)
	instanceCmd.AddCommand(instanceOrphanCmd)
	instanceCmd.AddCommand(instanceCopyCmd)

	return instanceCmd
}

func runInstanceGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	outputFormat, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	instanceID := args[0]

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	inst, err := mdClient.Instances.Get(ctx, instanceID)
	if err != nil {
		return err
	}

	switch outputFormat {
	case "json":
		jsonBytes, marshalErr := json.MarshalIndent(inst, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal instance to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
	case "text":
		if err := renderInstance(inst); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}

//nolint:dupl // parallel template-render shape with renderDeployment; consolidating would couple unrelated commands
func renderInstance(inst *types.Instance) error {
	tmplBytes, err := instanceTemplates.ReadFile("templates/instance.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("instance").Funcs(cli.MarkdownTemplateFuncs).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	paramsJSON := "{}"
	if inst.Params != nil {
		if b, marshalErr := json.MarshalIndent(inst.Params, "", "  "); marshalErr == nil {
			paramsJSON = string(b)
		}
	}

	data := struct {
		*types.Instance
		ParamsJSON string
	}{Instance: inst, ParamsJSON: paramsJSON}

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

func runInstanceDeploy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	name := args[0]

	action := deployments.ActionProvision
	if cmd.Name() == "destroy" {
		action = deployments.ActionDecommission
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

	if action == deployments.ActionDecommission {
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
	}

	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	if _, err = instance.RunDeploy(ctx, instance.NewDeployAPI(mdClient), name, opts); err != nil {
		return err
	}

	if action == deployments.ActionDecommission {
		fmt.Printf("✅ Instance `%s` decommission started\n", name)
		fmt.Printf("🔗 %s\n", mdClient.URLs.Helper(ctx).InstanceURL(name))
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

func runInstanceCopy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	sourceID := args[0]
	destinationID := args[1]
	message, err := cmd.Flags().GetString("message")
	if err != nil {
		return err
	}
	overridesPath, err := cmd.Flags().GetString("overrides")
	if err != nil {
		return err
	}
	copySecrets, err := cmd.Flags().GetBool("copy-secrets")
	if err != nil {
		return err
	}
	copyRefs, err := cmd.Flags().GetBool("copy-remote-references")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	var overrides map[string]any
	if overridesPath != "" {
		overrides, err = readParams(overridesPath)
		if err != nil {
			return err
		}
	}

	mdClient, mdClientErr := massdriver.NewClient()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	input := instances.CopyInput{
		Overrides:            overrides,
		CopySecrets:          copySecrets,
		CopyRemoteReferences: copyRefs,
		Message:              message,
	}

	inst, err := mdClient.Instances.Copy(ctx, sourceID, destinationID, input)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Instance `%s` copied to `%s` — plan deployment created on the destination\n", sourceID, destinationID)
	fmt.Printf("🔗 %s\n", mdClient.URLs.Helper(ctx).InstanceURL(inst.ID))
	return nil
}

func runInstanceExport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	instanceID := args[0]

	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
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

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	updatedInstance, err := mdClient.Instances.Update(ctx, instanceID, instances.UpdateInput{Version: version})
	if err != nil {
		return err
	}

	fmt.Printf("✅ Instance `%s` version set successfully\n", updatedInstance.ID)
	fmt.Printf("🔗 %s\n", mdClient.URLs.Helper(ctx).InstanceURL(updatedInstance.ID))

	return nil
}

func runInstanceOrphan(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	name := args[0]

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}
	deleteState, err := cmd.Flags().GetBool("delete-state")
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	if !force {
		if deleteState {
			fmt.Printf("%s: This will orphan instance %s, resetting it to INITIALIZED and permanently deleting its Terraform/OpenTofu state files. The next deployment will provision from scratch and may duplicate any resources tracked by the prior state. This is irreversible.\n", prettylogs.Orange("WARNING"), name)
		} else {
			fmt.Printf("%s: This will orphan instance %s, resetting it to INITIALIZED and clearing all of its Terraform/OpenTofu state locks.\n", prettylogs.Orange("WARNING"), name)
		}
		fmt.Printf("Type '%s' to confirm orphan: ", name)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.TrimSpace(answer) != name {
			fmt.Println("Orphan cancelled.")
			return nil
		}
	}

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	orphaned, err := mdClient.Instances.Orphan(ctx, name, instances.OrphanInput{DeleteState: deleteState})
	if err != nil {
		return err
	}

	fmt.Printf("✅ Instance `%s` orphaned (status: %s)\n", orphaned.ID, orphaned.Status)
	fmt.Printf("🔗 %s\n", mdClient.URLs.Helper(ctx).InstanceURL(orphaned.ID))
	return nil
}

func runInstanceList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	environmentID := args[0]

	cmd.SilenceUsage = true

	mdClient, err := massdriver.NewClient()
	if err != nil {
		return fmt.Errorf("error initializing massdriver client: %w", err)
	}

	insts, err := mdClient.Instances.List(ctx, instances.ListInput{EnvironmentID: environmentID})
	if err != nil {
		return err
	}

	tbl := cli.NewTable("ID", "Name", "Bundle", "Status")

	for _, inst := range insts {
		componentName := ""
		if inst.Component != nil {
			componentName = inst.Component.Name
		}
		bundleName := ""
		if inst.Bundle != nil {
			bundleName = inst.Bundle.Name
		}
		tbl.AddRow(inst.ID, componentName, bundleName, inst.Status)
	}

	tbl.Print()

	return nil
}

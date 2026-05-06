package cmd

import (
	"context"
	"fmt"

	"github.com/massdriver-cloud/mass/docs/helpdocs"
	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/mass/internal/commands/component"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/spf13/cobra"
)

// NewCmdComponent returns a cobra command for managing components in a project's blueprint.
func NewCmdComponent() *cobra.Command {
	componentCmd := &cobra.Command{
		Use:     "component",
		Aliases: []string{"comp"},
		Short:   "Manage components in a project's blueprint",
		Long:    helpdocs.MustRender("component"),
	}

	componentAddCmd := &cobra.Command{
		Use:     "add <project-id> <bundle-oci-repo-name>",
		Short:   "Add a component to a project's blueprint",
		Example: `mass component add ecomm aws-rds-cluster --id db --name "Primary Database"`,
		Long:    helpdocs.MustRender("component/add"),
		Args:    cobra.ExactArgs(2),
		RunE:    runComponentAdd,
	}
	componentAddCmd.Flags().String("id", "", "Short identifier for this component (e.g., db). Max 20 chars, lowercase alphanumeric.")
	componentAddCmd.Flags().StringP("name", "n", "", "Display name (defaults to --id if not provided)")
	componentAddCmd.Flags().StringP("description", "d", "", "Optional description")
	componentAddCmd.Flags().StringToStringP("attributes", "a", nil, "Custom attributes for ABAC (e.g. -a priority=high,cost-center=engineering)")
	_ = componentAddCmd.MarkFlagRequired("id")

	componentUpdateCmd := &cobra.Command{
		Use:     "update <component-id>",
		Short:   "Update a component's name, description, or attributes",
		Example: `mass component update ecomm-db --name "Primary DB" -a priority=high`,
		Args:    cobra.ExactArgs(1),
		RunE:    runComponentUpdate,
	}
	componentUpdateCmd.Flags().StringP("name", "n", "", "New display name")
	componentUpdateCmd.Flags().StringP("description", "d", "", "New description")
	componentUpdateCmd.Flags().StringToStringP("attributes", "a", nil, "Replacement custom attributes (e.g. -a priority=high,cost-center=engineering)")

	componentRemoveCmd := &cobra.Command{
		Use:     "remove <component-id>",
		Aliases: []string{"rm"},
		Short:   "Remove a component from a project's blueprint",
		Example: `mass component remove ecomm-db`,
		Long:    helpdocs.MustRender("component/remove"),
		Args:    cobra.ExactArgs(1),
		RunE:    runComponentRemove,
	}

	componentLinkCmd := &cobra.Command{
		Use:     "link <from-component>.<from-field> <to-component>.<to-field>",
		Short:   "Link two components in a project's blueprint",
		Example: `mass component link ecomm-db.authentication ecomm-app.database --from-version ~1.0 --to-version ~2.0`,
		Long:    helpdocs.MustRender("component/link"),
		Args:    cobra.ExactArgs(2),
		RunE:    runComponentLink,
	}
	componentLinkCmd.Flags().String("from-version", "latest", "Version constraint for the source component")
	componentLinkCmd.Flags().String("to-version", "latest", "Version constraint for the destination component")

	componentUnlinkCmd := &cobra.Command{
		Use:     "unlink <from-component>.<from-field> <to-component>.<to-field>",
		Short:   "Remove a link between two components",
		Example: `mass component unlink ecomm-db.authentication ecomm-app.database`,
		Long:    helpdocs.MustRender("component/unlink"),
		Args:    cobra.ExactArgs(2),
		RunE:    runComponentUnlink,
	}

	componentCmd.AddCommand(componentAddCmd)
	componentCmd.AddCommand(componentUpdateCmd)
	componentCmd.AddCommand(componentRemoveCmd)
	componentCmd.AddCommand(componentLinkCmd)
	componentCmd.AddCommand(componentUnlinkCmd)

	return componentCmd
}

func runComponentAdd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	projectID := args[0]
	ociRepoName := args[1]

	shortID, err := cmd.Flags().GetString("id")
	if err != nil {
		return err
	}
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	description, err := cmd.Flags().GetString("description")
	if err != nil {
		return err
	}
	attrs, err := cmd.Flags().GetStringToString("attributes")
	if err != nil {
		return err
	}
	if name == "" {
		name = shortID
	}

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	input := api.AddComponentInput{
		Id:          shortID,
		Name:        name,
		Description: description,
		Attributes:  cli.AttributesToAnyMap(attrs),
	}
	comp, addErr := api.AddComponent(ctx, mdClient, projectID, ociRepoName, input)
	if addErr != nil {
		return addErr
	}

	fmt.Printf("✅ Component `%s` added to project `%s`\n", comp.ID, projectID)
	return nil
}

func runComponentUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	componentID := args[0]
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	description, err := cmd.Flags().GetString("description")
	if err != nil {
		return err
	}
	attrs, err := cmd.Flags().GetStringToString("attributes")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	current, getErr := api.GetComponent(ctx, mdClient, componentID)
	if getErr != nil {
		return fmt.Errorf("error getting component: %w", getErr)
	}

	if !cmd.Flags().Changed("name") {
		name = current.Name
	}
	if !cmd.Flags().Changed("description") {
		description = current.Description
	}
	var attributes map[string]any
	if cmd.Flags().Changed("attributes") {
		attributes = cli.AttributesToAnyMap(attrs)
	} else {
		attributes = cli.StringMapToAnyMap(current.Attributes)
	}

	input := api.UpdateComponentInput{
		Name:        name,
		Description: description,
		Attributes:  attributes,
	}

	updated, err := api.UpdateComponent(ctx, mdClient, componentID, input)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Component `%s` updated\n", updated.ID)
	return nil
}

func runComponentRemove(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	componentID := args[0]
	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	comp, err := api.RemoveComponent(ctx, mdClient, componentID)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Component `%s` removed\n", comp.ID)
	return nil
}

func runComponentLink(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	fromComponentID, fromField, err := component.ParseComponentField(args[0])
	if err != nil {
		return err
	}
	toComponentID, toField, err := component.ParseComponentField(args[1])
	if err != nil {
		return err
	}

	fromVersion, err := cmd.Flags().GetString("from-version")
	if err != nil {
		return err
	}
	toVersion, err := cmd.Flags().GetString("to-version")
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	input := api.LinkComponentsInput{
		FromComponentId: fromComponentID,
		FromField:       fromField,
		FromVersion:     fromVersion,
		ToComponentId:   toComponentID,
		ToField:         toField,
		ToVersion:       toVersion,
	}
	link, linkErr := api.LinkComponents(ctx, mdClient, input)
	if linkErr != nil {
		return linkErr
	}

	fmt.Printf("✅ Linked `%s.%s` → `%s.%s` (id: %s)\n", fromComponentID, link.FromField, toComponentID, link.ToField, link.ID)
	return nil
}

func runComponentUnlink(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	fromComponentID, fromField, err := component.ParseComponentField(args[0])
	if err != nil {
		return err
	}
	toComponentID, toField, err := component.ParseComponentField(args[1])
	if err != nil {
		return err
	}

	projectID, _, err := component.SplitComponentID(fromComponentID)
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	mdClient, mdClientErr := client.New()
	if mdClientErr != nil {
		return fmt.Errorf("error initializing massdriver client: %w", mdClientErr)
	}

	links, err := api.ListLinks(ctx, mdClient, projectID, &api.LinksFilter{
		FromComponentId: &api.IdFilter{Eq: fromComponentID},
		ToComponentId:   &api.IdFilter{Eq: toComponentID},
	})
	if err != nil {
		return err
	}

	target, err := component.FindLink(links, fromField, toField)
	if err != nil {
		return fmt.Errorf("no link found from `%s.%s` to `%s.%s`", fromComponentID, fromField, toComponentID, toField)
	}

	if _, err := api.UnlinkComponents(ctx, mdClient, target.ID); err != nil {
		return err
	}

	fmt.Printf("✅ Unlinked `%s.%s` → `%s.%s`\n", fromComponentID, fromField, toComponentID, toField)
	return nil
}

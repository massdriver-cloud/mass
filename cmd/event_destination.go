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
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/mass/internal/commands/eventdestination"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

//go:embed templates/event-destination.get.md.tmpl
var eventDestinationTemplates embed.FS

// NewCmdEvent returns a cobra command for the organization's events.
func NewCmdEvent() *cobra.Command {
	eventCmd := &cobra.Command{
		Use:     "event",
		Short:   "Events, event types, and event destinations",
		Long:    helpdocs.MustRender("event"),
		Aliases: []string{"events"},
	}

	eventCmd.AddCommand(newCmdEventDestination())
	eventCmd.AddCommand(newCmdEventType())
	return eventCmd
}

func newCmdEventDestination() *cobra.Command { //nolint:funlen // cobra command builders are necessarily long
	eventDestinationCmd := &cobra.Command{
		Use:     "destination",
		Short:   "Event destination management",
		Long:    helpdocs.MustRender("event/destination"),
		Aliases: []string{"destinations", "dest"},
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List event destinations",
		Long:    helpdocs.MustRender("event/destination/list"),
		Args:    cobra.NoArgs,
		RunE:    runEventDestinationList,
	}
	listCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	listCmd.Flags().StringArray("status", nil, "Only destinations with this status (ACTIVE or PAUSED); repeat for more")

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get an event destination",
		Long:  helpdocs.MustRender("event/destination/get"),
		Args:  cobra.ExactArgs(1),
		RunE:  runEventDestinationGet,
	}
	getCmd.Flags().StringP("output", "o", "text", "Output format (text or json)")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an event destination",
		Long:  helpdocs.MustRender("event/destination/create"),
		Args:  cobra.NoArgs,
		RunE:  runEventDestinationCreate,
	}
	addEventDestinationFlags(createCmd.Flags())
	createCmd.Flags().String("adapter", "HTTP", "How the destination receives events")

	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an event destination",
		Long:  helpdocs.MustRender("event/destination/update"),
		Args:  cobra.ExactArgs(1),
		RunE:  runEventDestinationUpdate,
	}
	addEventDestinationFlags(updateCmd.Flags())

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an event destination",
		Long:  helpdocs.MustRender("event/destination/delete"),
		Args:  cobra.ExactArgs(1),
		RunE:  runEventDestinationDelete,
	}
	deleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	pauseCmd := &cobra.Command{
		Use:   "pause <id>",
		Short: "Pause an event destination",
		Long:  helpdocs.MustRender("event/destination/pause"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			api, err := newEventDestinationAPI(cmd)
			if err != nil {
				return err
			}
			return eventdestination.RunPause(context.Background(), api, args[0], os.Stdout)
		},
	}

	resumeCmd := &cobra.Command{
		Use:   "resume <id>",
		Short: "Resume a paused event destination",
		Long:  helpdocs.MustRender("event/destination/resume"),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			api, err := newEventDestinationAPI(cmd)
			if err != nil {
				return err
			}
			return eventdestination.RunResume(context.Background(), api, args[0], os.Stdout)
		},
	}

	previewCmd := &cobra.Command{
		Use:   "preview",
		Short: "List recent events that event types, attributes, and a condition select",
		Long:  helpdocs.MustRender("event/destination/preview"),
		Args:  cobra.NoArgs,
		RunE:  runEventDestinationPreview,
	}
	previewCmd.Flags().StringArray("event-type", nil, "Event type, or <entity>.* for every event of an entity; repeat for more")
	previewCmd.Flags().StringArray("attribute", nil, "Attribute the event's entity must have had, key=value; repeat for more")
	previewCmd.Flags().String("condition", "", "jq expression that must output true")
	previewCmd.Flags().Int("limit", 25, "Most events to list")
	previewCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")

	deliveriesCmd := &cobra.Command{
		Use:   "deliveries <id>",
		Short: "List an event destination's deliveries",
		Long:  helpdocs.MustRender("event/destination/deliveries"),
		Args:  cobra.ExactArgs(1),
		RunE:  runEventDestinationDeliveries,
	}
	deliveriesCmd.Flags().StringArray("status", nil, "Only deliveries with this status (DELIVERED, SKIPPED, FAILED, or PENDING); repeat for more")
	deliveriesCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")

	redeliverCmd := &cobra.Command{
		Use:   "redeliver <id>",
		Short: "Send failed deliveries again",
		Long:  helpdocs.MustRender("event/destination/redeliver"),
		Args:  cobra.ExactArgs(1),
		RunE:  runEventDestinationRedeliver,
	}
	redeliverCmd.Flags().StringArray("delivery", nil, "Failed delivery id to send again; repeat for more. Leave out to send every failed delivery")

	eventDestinationCmd.AddCommand(listCmd)
	eventDestinationCmd.AddCommand(getCmd)
	eventDestinationCmd.AddCommand(createCmd)
	eventDestinationCmd.AddCommand(updateCmd)
	eventDestinationCmd.AddCommand(deleteCmd)
	eventDestinationCmd.AddCommand(pauseCmd)
	eventDestinationCmd.AddCommand(resumeCmd)
	eventDestinationCmd.AddCommand(previewCmd)
	eventDestinationCmd.AddCommand(deliveriesCmd)
	eventDestinationCmd.AddCommand(redeliverCmd)

	return eventDestinationCmd
}

func addEventDestinationFlags(flags *pflag.FlagSet) {
	flags.String("name", "", "Name of the destination, unique in the organization")
	flags.String("config", "", "Adapter config as JSON, or @file")
	flags.String("auth", "", "Adapter credentials as JSON, or @file. Never returned by the API")
	flags.String("url", "", "URL to post events to (sets url in the HTTP config)")
	flags.StringArray("header", nil, "Header sent with each event, Name=value (sets headers in the HTTP config, which the API returns; put credentials in --auth); repeat for more")
	flags.StringArray("event-type", nil, "Event type, or <entity>.* for every event of an entity; repeat for more")
	flags.StringArray("attribute", nil, "Attribute the event's entity must have had, key=value; repeat for more")
	flags.String("condition", "", "jq expression that must output true for the event to be sent")
}

func eventDestinationOptions(flags *pflag.FlagSet) eventdestination.Options {
	opts := eventdestination.Options{}
	opts.Name, _ = flags.GetString("name")
	opts.Adapter, _ = flags.GetString("adapter")
	opts.Config, _ = flags.GetString("config")
	opts.Auth, _ = flags.GetString("auth")
	opts.URL, _ = flags.GetString("url")
	opts.Headers, _ = flags.GetStringArray("header")
	opts.EventTypes, _ = flags.GetStringArray("event-type")
	opts.Attributes, _ = flags.GetStringArray("attribute")
	opts.Condition, _ = flags.GetString("condition")

	opts.NameSet = flags.Changed("name")
	opts.ConfigSet = flags.Changed("config")
	opts.AuthSet = flags.Changed("auth")
	opts.URLSet = flags.Changed("url")
	opts.HeadersSet = flags.Changed("header")
	opts.EventTypesSet = flags.Changed("event-type")
	opts.AttributesSet = flags.Changed("attribute")
	opts.ConditionSet = flags.Changed("condition")
	return opts
}

func newEventDestinationAPI(cmd *cobra.Command) (eventdestination.API, error) {
	mdClient, err := newMassdriverClient(cmd)
	if err != nil {
		return nil, fmt.Errorf("error initializing massdriver client: %w", err)
	}
	return eventdestination.NewAPI(mdClient), nil
}

func runEventDestinationList(cmd *cobra.Command, _ []string) error {
	output, _ := cmd.Flags().GetString("output")
	statuses, _ := cmd.Flags().GetStringArray("status")
	cmd.SilenceUsage = true

	api, err := newEventDestinationAPI(cmd)
	if err != nil {
		return err
	}
	return eventdestination.RunList(context.Background(), api, statuses, output, os.Stdout)
}

func runEventDestinationGet(cmd *cobra.Command, args []string) error {
	output, _ := cmd.Flags().GetString("output")
	cmd.SilenceUsage = true

	api, err := newEventDestinationAPI(cmd)
	if err != nil {
		return err
	}
	destination, err := api.Get(context.Background(), args[0])
	if err != nil {
		return err
	}

	switch output {
	case "json":
		jsonBytes, marshalErr := json.MarshalIndent(destination, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal event destination to JSON: %w", marshalErr)
		}
		fmt.Println(string(jsonBytes))
		return nil
	case "text":
		return renderEventDestination(destination)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func renderEventDestination(d *types.EventDestination) error {
	tmplBytes, err := eventDestinationTemplates.ReadFile("templates/event-destination.get.md.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}
	tmpl, err := template.New("event-destination").Funcs(cli.MarkdownTemplateFuncs).Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	config, err := json.MarshalIndent(d.Config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	data := struct {
		*types.EventDestination
		ConfigJSON string
	}{d, string(config)}

	var buf bytes.Buffer
	if execErr := tmpl.Execute(&buf, data); execErr != nil {
		return fmt.Errorf("failed to execute template: %w", execErr)
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

func runEventDestinationCreate(cmd *cobra.Command, _ []string) error {
	opts := eventDestinationOptions(cmd.Flags())
	cmd.SilenceUsage = true

	api, err := newEventDestinationAPI(cmd)
	if err != nil {
		return err
	}
	return eventdestination.RunCreate(context.Background(), api, opts, os.Stdout)
}

func runEventDestinationUpdate(cmd *cobra.Command, args []string) error {
	opts := eventDestinationOptions(cmd.Flags())
	cmd.SilenceUsage = true

	api, err := newEventDestinationAPI(cmd)
	if err != nil {
		return err
	}
	return eventdestination.RunUpdate(context.Background(), api, args[0], opts, os.Stdout)
}

func runEventDestinationDelete(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	cmd.SilenceUsage = true

	api, err := newEventDestinationAPI(cmd)
	if err != nil {
		return err
	}
	return eventdestination.RunDelete(context.Background(), api, args[0], force, os.Stdin, os.Stdout)
}

func runEventDestinationPreview(cmd *cobra.Command, _ []string) error {
	opts := eventdestination.PreviewOptions{}
	opts.EventTypes, _ = cmd.Flags().GetStringArray("event-type")
	opts.Attributes, _ = cmd.Flags().GetStringArray("attribute")
	opts.Condition, _ = cmd.Flags().GetString("condition")
	opts.Limit, _ = cmd.Flags().GetInt("limit")
	opts.Output, _ = cmd.Flags().GetString("output")
	cmd.SilenceUsage = true

	api, err := newEventDestinationAPI(cmd)
	if err != nil {
		return err
	}
	return eventdestination.RunPreview(context.Background(), api, opts, os.Stdout)
}

func runEventDestinationDeliveries(cmd *cobra.Command, args []string) error {
	statuses, _ := cmd.Flags().GetStringArray("status")
	output, _ := cmd.Flags().GetString("output")
	cmd.SilenceUsage = true

	api, err := newEventDestinationAPI(cmd)
	if err != nil {
		return err
	}
	return eventdestination.RunDeliveries(context.Background(), api, args[0], statuses, output, os.Stdout)
}

func runEventDestinationRedeliver(cmd *cobra.Command, args []string) error {
	deliveryIDs, _ := cmd.Flags().GetStringArray("delivery")
	cmd.SilenceUsage = true

	api, err := newEventDestinationAPI(cmd)
	if err != nil {
		return err
	}
	return eventdestination.RunRedeliver(context.Background(), api, args[0], deliveryIDs, os.Stdout)
}

func newCmdEventType() *cobra.Command {
	eventTypeCmd := &cobra.Command{
		Use:     "type",
		Short:   "Event catalog",
		Long:    helpdocs.MustRender("event/type"),
		Aliases: []string{"types"},
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List event types",
		Long:    helpdocs.MustRender("event/type/list"),
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			entity, _ := cmd.Flags().GetString("entity")
			output, _ := cmd.Flags().GetString("output")
			cmd.SilenceUsage = true

			api, err := newEventDestinationAPI(cmd)
			if err != nil {
				return err
			}
			return eventdestination.RunEventTypeList(context.Background(), api, entity, output, os.Stdout)
		},
	}
	listCmd.Flags().String("entity", "", "Only event types of this entity, e.g. deployment")
	listCmd.Flags().StringP("output", "o", "table", "Output format (table, json)")

	eventTypeCmd.AddCommand(listCmd)
	return eventTypeCmd
}

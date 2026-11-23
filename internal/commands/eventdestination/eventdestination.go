// Package eventdestination holds the logic behind `mass event-destination`
// and `mass event-type`.
package eventdestination

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/eventdestinations"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// API is the narrow SDK surface the event destination commands use.
type API interface {
	Get(ctx context.Context, id string) (*types.EventDestination, error)
	Iter(ctx context.Context, input eventdestinations.ListInput) iter.Seq2[types.EventDestination, error]
	Create(ctx context.Context, input eventdestinations.CreateInput) (*types.EventDestination, error)
	Update(ctx context.Context, id string, input eventdestinations.UpdateInput) (*types.EventDestination, error)
	Delete(ctx context.Context, id string) (*types.EventDestination, error)
	Pause(ctx context.Context, id string) (*types.EventDestination, error)
	Resume(ctx context.Context, id string) (*types.EventDestination, error)
	Redeliver(ctx context.Context, id string, deliveryIDs []string) (*types.EventDestination, error)
	IterDeliveries(ctx context.Context, id string, input eventdestinations.ListDeliveriesInput) iter.Seq2[types.EventDelivery, error]
	IterPreview(ctx context.Context, input eventdestinations.PreviewInput) iter.Seq2[types.AuditLog, error]
	ListEventTypes(ctx context.Context, input eventdestinations.ListEventTypesInput) ([]types.EventType, error)
}

// NewAPI returns the production [API] backed by the SDK client.
func NewAPI(c *massdriver.Client) API { return c.EventDestinations }

const timeFormat = "2006-01-02 15:04:05"

// Options holds the flags shared by create and update. The *Set fields say
// whether a flag was given, since update sends only the fields that were.
type Options struct {
	Name       string
	Adapter    string
	Config     string
	Auth       string
	URL        string
	Headers    []string
	EventTypes []string
	Attributes []string
	Condition  string

	NameSet       bool
	ConfigSet     bool
	AuthSet       bool
	URLSet        bool
	HeadersSet    bool
	EventTypesSet bool
	AttributesSet bool
	ConditionSet  bool
}

// RunCreate creates a destination and prints its signing secret, which the
// receiver needs to check signatures.
func RunCreate(ctx context.Context, api API, opts Options, out io.Writer) error {
	if opts.Name == "" {
		return errors.New("--name is required")
	}
	config, err := buildConfig(nil, opts)
	if err != nil {
		return err
	}
	if len(config) == 0 {
		return errors.New("set --url, --header, or --config")
	}
	input := eventdestinations.CreateInput{
		Name:       opts.Name,
		Adapter:    strings.ToUpper(opts.Adapter),
		Config:     config,
		EventTypes: nonEmpty(opts.EventTypes),
		Condition:  opts.Condition,
	}
	if opts.AuthSet {
		if input.Auth, err = ParseJSONObject(opts.Auth, "--auth"); err != nil {
			return err
		}
	}
	if input.Attributes, err = ParsePairs(opts.Attributes, "--attribute"); err != nil {
		return err
	}

	created, err := api.Create(ctx, input)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "✅ Event destination `%s` created (ID: %s)\n", created.Name, created.ID)
	if created.SigningSecret != "" {
		fmt.Fprintf(out, "🔑 Signing secret: %s\n", created.SigningSecret)
		fmt.Fprintln(out, "   Store it now. Your receiver uses it to check the webhook-signature header.")
	}
	return nil
}

// RunUpdate changes the fields whose flags were given. --url and --header
// change those keys of the destination's current config, because a new
// config replaces the old one whole.
func RunUpdate(ctx context.Context, api API, id string, opts Options, out io.Writer) error {
	configSet := opts.ConfigSet || opts.URLSet || opts.HeadersSet
	if !opts.NameSet && !configSet && !opts.AuthSet && !opts.EventTypesSet && !opts.AttributesSet && !opts.ConditionSet {
		return errors.New("nothing to update: set at least one of --name, --config, --url, --header, --auth, --event-type, --attribute, or --condition")
	}

	input, err := updateInput(opts)
	if err != nil {
		return err
	}
	if configSet {
		var base map[string]any
		if !opts.ConfigSet {
			current, getErr := api.Get(ctx, id)
			if getErr != nil {
				return getErr
			}
			base = current.Config
		}
		if input.Config, err = buildConfig(base, opts); err != nil {
			return err
		}
	}

	updated, err := api.Update(ctx, id, input)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "✅ Event destination `%s` updated\n", updated.Name)
	return nil
}

func updateInput(opts Options) (eventdestinations.UpdateInput, error) {
	input := eventdestinations.UpdateInput{}
	if opts.NameSet {
		input.Name = &opts.Name
	}
	if opts.AuthSet {
		auth, err := ParseJSONObject(opts.Auth, "--auth")
		if err != nil {
			return input, err
		}
		input.Auth = auth
	}
	if opts.EventTypesSet {
		eventTypes := append([]string{}, nonEmpty(opts.EventTypes)...)
		input.EventTypes = &eventTypes
	}
	if opts.AttributesSet {
		attributes, err := ParsePairs(opts.Attributes, "--attribute")
		if err != nil {
			return input, err
		}
		input.Attributes = attributes
	}
	if opts.ConditionSet {
		input.Condition = &opts.Condition
	}
	return input, nil
}

// RunDelete asks for the destination's name before deleting it, unless
// force is set, because deleting drops the events that wait to be sent.
func RunDelete(ctx context.Context, api API, id string, force bool, in io.Reader, out io.Writer) error {
	destination, err := api.Get(ctx, id)
	if err != nil {
		return err
	}
	if !force {
		fmt.Fprintf(out, "WARNING: This deletes event destination `%s` and its deliveries. Events that wait to be sent are not sent.\n", destination.Name)
		fmt.Fprintf(out, "Type `%s` to confirm deletion: ", destination.Name)
		answer, _ := bufio.NewReader(in).ReadString('\n')
		if strings.TrimSpace(answer) != destination.Name {
			fmt.Fprintln(out, "Deletion cancelled.")
			return nil
		}
	}
	if _, err := api.Delete(ctx, id); err != nil {
		return err
	}
	fmt.Fprintf(out, "Event destination `%s` deleted\n", destination.Name)
	return nil
}

// RunPause pauses a destination.
func RunPause(ctx context.Context, api API, id string, out io.Writer) error {
	destination, err := api.Pause(ctx, id)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "⏸️  Event destination `%s` paused. Events wait until it is resumed.\n", destination.Name)
	return nil
}

// RunResume resumes a destination.
func RunResume(ctx context.Context, api API, id string, out io.Writer) error {
	destination, err := api.Resume(ctx, id)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "▶️  Event destination `%s` resumed\n", destination.Name)
	return nil
}

// RunRedeliver sends failed deliveries again: the given ones, or every
// failed delivery when deliveryIDs is empty.
func RunRedeliver(ctx context.Context, api API, id string, deliveryIDs []string, out io.Writer) error {
	destination, err := api.Redeliver(ctx, id, deliveryIDs)
	if err != nil {
		return err
	}
	if len(deliveryIDs) == 0 {
		fmt.Fprintf(out, "🔁 Every failed delivery of `%s` is queued to send again\n", destination.Name)
		return nil
	}
	fmt.Fprintf(out, "🔁 %d failed deliveries of `%s` are queued to send again\n", len(deliveryIDs), destination.Name)
	return nil
}

var destinationStatuses = []string{string(eventdestinations.StatusActive), string(eventdestinations.StatusPaused)}

// RunList lists destinations, optionally only those with the given statuses.
func RunList(ctx context.Context, api API, statuses []string, output string, out *os.File) error {
	parsed, err := parseStatuses(statuses, destinationStatuses)
	if err != nil {
		return err
	}
	input := eventdestinations.ListInput{}
	for _, s := range parsed {
		input.Statuses = append(input.Statuses, eventdestinations.Status(s))
	}
	return render(api.Iter(ctx, input), output, out, cli.PagerConfig[types.EventDestination]{
		Columns: []string{"ID", "Name", "Adapter", "Status", "Event Types", "Failing Since", "Last Error"},
		Row: func(d types.EventDestination) []string {
			return []string{d.ID, d.Name, d.Adapter, d.Status, eventTypesCell(d.EventTypes), formatTime(d.FailingSince), cli.TruncateString(d.LastError, 40)}
		},
	})
}

var deliveryStatuses = []string{
	string(eventdestinations.DeliveryDelivered),
	string(eventdestinations.DeliverySkipped),
	string(eventdestinations.DeliveryFailed),
	string(eventdestinations.DeliveryPending),
}

// RunDeliveries lists a destination's deliveries, newest first.
func RunDeliveries(ctx context.Context, api API, id string, statuses []string, output string, out *os.File) error {
	parsed, err := parseStatuses(statuses, deliveryStatuses)
	if err != nil {
		return err
	}
	input := eventdestinations.ListDeliveriesInput{}
	for _, s := range parsed {
		input.Statuses = append(input.Statuses, eventdestinations.DeliveryStatus(s))
	}
	return render(api.IterDeliveries(ctx, id, input), output, out, cli.PagerConfig[types.EventDelivery]{
		Columns: []string{"ID", "Status", "Event Type", "Attempts", "Created At", "Delivered At", "Last Error"},
		Row: func(d types.EventDelivery) []string {
			eventType := ""
			if d.Event != nil {
				eventType = d.Event.Type
			}
			return []string{d.ID, d.Status, eventType, strconv.Itoa(d.Attempts), formatTime(d.CreatedAt), formatTime(d.DeliveredAt), cli.TruncateString(d.LastError, 40)}
		},
	})
}

// PreviewOptions holds the flags of `mass event-destination preview`.
type PreviewOptions struct {
	EventTypes []string
	Attributes []string
	Condition  string
	Limit      int
	Output     string
}

// RunPreview lists up to opts.Limit recent audit log events that the
// selection matches, newest first. The limit keeps an empty selection from
// paging through the whole audit log.
func RunPreview(ctx context.Context, api API, opts PreviewOptions, out *os.File) error {
	attributes, err := ParsePairs(opts.Attributes, "--attribute")
	if err != nil {
		return err
	}
	seq := api.IterPreview(ctx, eventdestinations.PreviewInput{
		EventTypes: nonEmpty(opts.EventTypes),
		Attributes: attributes,
		Condition:  opts.Condition,
		PageSize:   min(opts.Limit, 100),
	})
	return render(take(seq, opts.Limit), opts.Output, out, cli.PagerConfig[types.AuditLog]{
		Columns: []string{"ID", "Type", "Occurred At", "Subject"},
		Row: func(e types.AuditLog) []string {
			return []string{e.ID, e.Type, formatTime(e.OccurredAt), e.Subject}
		},
	})
}

// RunEventTypeList lists the event catalog, optionally for one entity.
func RunEventTypeList(ctx context.Context, api API, entity, output string, out *os.File) error {
	eventTypes, err := api.ListEventTypes(ctx, eventdestinations.ListEventTypesInput{Entity: entity})
	if err != nil {
		return err
	}
	return render(seqOf(eventTypes), output, out, cli.PagerConfig[types.EventType]{
		Columns: []string{"ID", "Entity", "Change", "Description"},
		Row: func(e types.EventType) []string {
			return []string{e.ID, e.Entity, e.ChangeType, cli.TruncateString(e.Description, 80)}
		},
	})
}

// ParsePairs parses repeated key=value flags. Empty values are dropped, so
// a flag given only an empty value gives an empty map.
func ParsePairs(pairs []string, flag string) (map[string]string, error) {
	var out map[string]string
	for _, pair := range pairs {
		if pair == "" {
			if out == nil {
				out = map[string]string{}
			}
			continue
		}
		key, value, found := strings.Cut(pair, "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return nil, fmt.Errorf("%s %q must be formatted as key=value", flag, pair)
		}
		if out == nil {
			out = map[string]string{}
		}
		out[key] = value
	}
	return out, nil
}

// ParseJSONObject reads a JSON object from s, or from the file s names
// when it starts with @. The JSON error is not wrapped, because for --auth
// it can quote the credentials.
func ParseJSONObject(s, flag string) (map[string]any, error) {
	data := []byte(s)
	if path, isFile := strings.CutPrefix(s, "@"); isFile {
		fileData, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", flag, err)
		}
		data = fileData
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil || obj == nil {
		return nil, fmt.Errorf("%s must be a JSON object or @file holding one", flag)
	}
	return obj, nil
}

func buildConfig(base map[string]any, opts Options) (map[string]any, error) {
	config := map[string]any{}
	for k, v := range base {
		config[k] = v
	}
	if opts.ConfigSet {
		parsed, err := ParseJSONObject(opts.Config, "--config")
		if err != nil {
			return nil, err
		}
		config = parsed
	}
	if opts.URLSet {
		config["url"] = opts.URL
	}
	if opts.HeadersSet {
		headers, err := ParsePairs(opts.Headers, "--header")
		if err != nil {
			return nil, err
		}
		merged := map[string]any{}
		if current, ok := config["headers"].(map[string]any); ok {
			for k, v := range current {
				merged[k] = v
			}
		}
		for k, v := range headers {
			merged[k] = v
		}
		config["headers"] = merged
	}
	return config, nil
}

func parseStatuses(given, allowed []string) ([]string, error) {
	out := make([]string, 0, len(given))
	for _, s := range given {
		upper := strings.ToUpper(s)
		if !slices.Contains(allowed, upper) {
			return nil, fmt.Errorf("--status %q must be one of %s", s, strings.Join(allowed, ", "))
		}
		out = append(out, upper)
	}
	return out, nil
}

func nonEmpty(values []string) []string {
	var out []string
	for _, v := range values {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func render[T any](seq iter.Seq2[T, error], output string, out *os.File, cfg cli.PagerConfig[T]) error {
	switch output {
	case "json":
		items, err := types.Collect(seq)
		if err != nil {
			return err
		}
		if items == nil {
			items = []T{}
		}
		jsonBytes, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal to JSON: %w", err)
		}
		fmt.Fprintln(out, string(jsonBytes))
		return nil
	case "table":
		cfg.Out = out
		return cli.Paginate(seq, cfg)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func take[T any](seq iter.Seq2[T, error], n int) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		if n <= 0 {
			return
		}
		count := 0
		for v, err := range seq {
			if !yield(v, err) || err != nil {
				return
			}
			count++
			if count == n {
				return
			}
		}
	}
}

func seqOf[T any](items []T) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for _, item := range items {
			if !yield(item, nil) {
				return
			}
		}
	}
}

func eventTypesCell(eventTypes []string) string {
	if len(eventTypes) == 0 {
		return "every event"
	}
	return cli.TruncateString(strings.Join(eventTypes, ", "), 40)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(timeFormat)
}

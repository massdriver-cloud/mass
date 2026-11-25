// Package stash holds the logic behind `mass stash`.
package stash

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
	sdkstash "github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/stash"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// API is the narrow SDK surface the stash commands use.
type API interface {
	Get(ctx context.Context, scope sdkstash.Scope, key string) (*types.StashEntry, error)
	Iter(ctx context.Context, scope sdkstash.Scope, input sdkstash.ListInput) iter.Seq2[types.StashEntry, error]
	Set(ctx context.Context, scope sdkstash.Scope, key string, input sdkstash.SetInput) (*types.StashEntry, error)
	Delete(ctx context.Context, scope sdkstash.Scope, key string) error
	EnvironmentExists(ctx context.Context, id string) (bool, error)
	ComponentExists(ctx context.Context, id string) (bool, error)
}

type sdkAPI struct {
	*sdkstash.Service

	client *massdriver.Client
}

// NewAPI returns the production [API] backed by the SDK client.
func NewAPI(c *massdriver.Client) API { return sdkAPI{Service: c.Stash, client: c} }

func (a sdkAPI) EnvironmentExists(ctx context.Context, id string) (bool, error) {
	_, err := a.client.Environments.Get(ctx, id)
	return exists(err)
}

func (a sdkAPI) ComponentExists(ctx context.Context, id string) (bool, error) {
	_, err := a.client.Components.Get(ctx, id)
	return exists(err)
}

func exists(err error) (bool, error) {
	if errors.Is(err, gql.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

var (
	scopeIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	keyPattern     = regexp.MustCompile(`^[a-zA-Z0-9_./-]+$`)
	uuidPattern    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

var scopePrefixes = map[string]sdkstash.ScopeKind{
	"project":     sdkstash.ScopeProject,
	"env":         sdkstash.ScopeEnvironment,
	"environment": sdkstash.ScopeEnvironment,
	"component":   sdkstash.ScopeComponent,
	"instance":    sdkstash.ScopeInstance,
	"deployment":  sdkstash.ScopeDeployment,
}

// ParseAddress splits "<scope>.<key>" at the first ".". Scope identifiers
// never contain ".", so the key keeps any later dots.
func ParseAddress(address string) (scope, key string, err error) {
	scope, key, found := strings.Cut(address, ".")
	if !found || scope == "" || key == "" {
		return "", "", fmt.Errorf("invalid address %q: expected <scope>.<key>", address)
	}
	if !keyPattern.MatchString(key) {
		return "", "", fmt.Errorf("invalid key %q: use letters, digits, and _ . / -", key)
	}
	return scope, key, nil
}

// ResolveScope turns a scope reference into a stash scope. A "<kind>:" prefix
// names the kind. Without one, the identifier's shape decides: a UUID is a
// deployment, one segment is a project, and three are an instance. Two
// segments fit both an environment and a component, so the API is asked which
// one exists.
func ResolveScope(ctx context.Context, api API, ref string) (sdkstash.Scope, error) {
	if prefix, id, ok := strings.Cut(ref, ":"); ok {
		kind, known := scopePrefixes[prefix]
		if !known {
			return sdkstash.Scope{}, fmt.Errorf("unknown scope prefix %q: use project, env, environment, component, instance, or deployment", prefix)
		}
		if !scopeIDPattern.MatchString(id) {
			return sdkstash.Scope{}, fmt.Errorf("invalid %s identifier %q", kind, id)
		}
		return sdkstash.Scope{Kind: kind, ID: id}, nil
	}

	if uuidPattern.MatchString(ref) {
		return sdkstash.Scope{Kind: sdkstash.ScopeDeployment, ID: ref}, nil
	}
	if !scopeIDPattern.MatchString(ref) {
		return sdkstash.Scope{}, fmt.Errorf("invalid scope %q: use lowercase letters, digits, and -", ref)
	}

	switch strings.Count(ref, "-") + 1 {
	case 1:
		return sdkstash.Scope{Kind: sdkstash.ScopeProject, ID: ref}, nil
	case 2:
		return resolveEnvironmentOrComponent(ctx, api, ref)
	case 3:
		return sdkstash.Scope{Kind: sdkstash.ScopeInstance, ID: ref}, nil
	}
	return sdkstash.Scope{}, fmt.Errorf("invalid scope %q: expected a project, environment, component, instance, or deployment identifier", ref)
}

func resolveEnvironmentOrComponent(ctx context.Context, api API, ref string) (sdkstash.Scope, error) {
	isEnv, err := api.EnvironmentExists(ctx, ref)
	if err != nil {
		return sdkstash.Scope{}, err
	}
	isComp, err := api.ComponentExists(ctx, ref)
	if err != nil {
		return sdkstash.Scope{}, err
	}
	switch {
	case isEnv && isComp:
		return sdkstash.Scope{}, fmt.Errorf("%s matches an environment and a component; use env:%s or component:%s", ref, ref, ref)
	case isEnv:
		return sdkstash.Scope{Kind: sdkstash.ScopeEnvironment, ID: ref}, nil
	case isComp:
		return sdkstash.Scope{Kind: sdkstash.ScopeComponent, ID: ref}, nil
	}
	return sdkstash.Scope{}, fmt.Errorf("no environment or component %s: %w", ref, gql.ErrNotFound)
}

func resolveAddress(ctx context.Context, api API, address string) (sdkstash.Scope, string, error) {
	ref, key, err := ParseAddress(address)
	if err != nil {
		return sdkstash.Scope{}, "", err
	}
	scope, err := ResolveScope(ctx, api, ref)
	if err != nil {
		return sdkstash.Scope{}, "", err
	}
	return scope, key, nil
}

// RunGet prints an entry's value exactly as stored, so `$(mass stash get ...)`
// captures it. With output "json" it prints the whole entry.
func RunGet(ctx context.Context, api API, address, output string, out io.Writer) error {
	scope, key, err := resolveAddress(ctx, api, address)
	if err != nil {
		return err
	}
	entry, err := api.Get(ctx, scope, key)
	if err != nil {
		return err
	}

	switch output {
	case "text":
		if entry.Value == nil {
			return fmt.Errorf("%s is sensitive: reading it needs write access to %s", key, scope)
		}
		_, err = io.WriteString(out, *entry.Value)
		return err
	case "json":
		return writeJSON(out, entry)
	}
	return fmt.Errorf("unsupported output format: %s", output)
}

// SetOptions holds the flags of `mass stash set`.
type SetOptions struct {
	Label     string
	Format    string
	TTL       int
	Immutable bool
	Sensitive bool
}

// RunSet stores value under the address. A value of "-" reads it from in.
func RunSet(ctx context.Context, api API, address, value string, opts SetOptions, in io.Reader, out io.Writer) error {
	input := sdkstash.SetInput{Label: opts.Label, Immutable: opts.Immutable, Sensitive: opts.Sensitive}

	switch strings.ToLower(opts.Format) {
	case "", "text":
		input.Format = sdkstash.FormatText
	case "json":
		input.Format = sdkstash.FormatJSON
	default:
		return fmt.Errorf("invalid --format %q: use text or json", opts.Format)
	}
	if opts.TTL < 0 {
		return fmt.Errorf("invalid --ttl %d: use zero or more seconds", opts.TTL)
	}
	input.TTL = time.Duration(opts.TTL) * time.Second

	if value == "-" {
		data, err := io.ReadAll(in)
		if err != nil {
			return fmt.Errorf("read value from stdin: %w", err)
		}
		value = string(data)
	}
	if input.Format == sdkstash.FormatJSON {
		var obj map[string]any
		if err := json.Unmarshal([]byte(value), &obj); err != nil || obj == nil {
			return errors.New("--format json needs the value to be a JSON object")
		}
	}
	input.Value = value

	scope, key, err := resolveAddress(ctx, api, address)
	if err != nil {
		return err
	}
	if _, err := api.Set(ctx, scope, key, input); err != nil {
		return err
	}
	fmt.Fprintf(out, "✅ Stashed `%s` in %s `%s`\n", key, scope.Kind, scope.ID)
	return nil
}

// RunDelete asks for the key before deleting the entry, unless force is set.
func RunDelete(ctx context.Context, api API, address string, force bool, in io.Reader, out io.Writer) error {
	scope, key, err := resolveAddress(ctx, api, address)
	if err != nil {
		return err
	}
	if !force {
		if _, err := api.Get(ctx, scope, key); err != nil {
			return err
		}
		fmt.Fprintf(out, "Type `%s` to delete it from %s `%s`: ", key, scope.Kind, scope.ID)
		answer, readErr := bufio.NewReader(in).ReadString('\n')
		if readErr != nil && answer == "" {
			return errors.New("no confirmation given; set --force to delete without one")
		}
		if strings.TrimSpace(answer) != key {
			fmt.Fprintln(out, "Deletion cancelled.")
			return nil
		}
	}
	if err := api.Delete(ctx, scope, key); err != nil {
		return err
	}
	fmt.Fprintf(out, "Deleted `%s` from %s `%s`\n", key, scope.Kind, scope.ID)
	return nil
}

const timeFormat = "2006-01-02 15:04:05"

// RunList lists the entries in one scope.
func RunList(ctx context.Context, api API, ref, output string, out *os.File) error {
	scope, err := ResolveScope(ctx, api, ref)
	if err != nil {
		return err
	}
	seq := api.Iter(ctx, scope, sdkstash.ListInput{})

	switch output {
	case "json":
		entries, err := types.Collect(seq)
		if err != nil {
			return err
		}
		if entries == nil {
			entries = []types.StashEntry{}
		}
		return writeJSON(out, entries)
	case "table":
		return cli.Paginate(seq, cli.PagerConfig[types.StashEntry]{
			Columns: []string{"Key", "Label", "Value", "Format", "Flags", "Expires", "Updated"},
			Row: func(e types.StashEntry) []string {
				return []string{e.Key, cli.TruncateString(e.Label, 40), valueCell(e), e.Format, flagsCell(e), formatTime(e.ExpiresAt), formatTime(e.UpdatedAt)}
			},
			Out: out,
		})
	}
	return fmt.Errorf("unsupported output format: %s", output)
}

func valueCell(e types.StashEntry) string {
	if e.Value == nil {
		return "(sensitive)"
	}
	return cli.TruncateString(strings.ReplaceAll(*e.Value, "\n", `\n`), 60)
}

func flagsCell(e types.StashEntry) string {
	var flags []string
	if e.Immutable {
		flags = append(flags, "immutable")
	}
	if e.Sensitive {
		flags = append(flags, "sensitive")
	}
	return strings.Join(flags, ",")
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format(timeFormat)
}

func writeJSON(out io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal to JSON: %w", err)
	}
	fmt.Fprintln(out, string(data))
	return nil
}

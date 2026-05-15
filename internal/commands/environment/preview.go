package environment

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/environments"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/instances"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
	"sigs.k8s.io/yaml"
)

// PreviewConfig is the YAML config that drives `mass environment preview`.
//
// Stable terminology:
//   - `project` (no "slug")
//   - `baseEnvironment` (the env we're forking from)
//   - `environmentDefaults` use `resourceType` (catalog name) + `resourceId` — no
//     "artifact" terminology, no "massdriver/" scoping prefix.
//   - `instances` (V2 term; not "packages")
type PreviewConfig struct {
	// Project is the project identifier the preview env lives in. Required.
	Project string `json:"project"`

	// BaseEnvironment is the identifier (local segment) of the env we fork from. Required.
	BaseEnvironment string `json:"baseEnvironment"`

	// CopyEnvironmentDefaults inherits the parent's default resource connections
	// into the fork on top of any explicit `environmentDefaults` overrides.
	CopyEnvironmentDefaults bool `json:"copyEnvironmentDefaults,omitempty"`

	// CopySecrets fans copyInstance's `copySecrets: true` across every package
	// during the fork. Per-instance secret overrides in `instances` still apply
	// after this.
	CopySecrets bool `json:"copySecrets,omitempty"`

	// CopyRemoteReferences fans copyInstance's `copyRemoteReferences: true`
	// across every package during the fork. The SDK does not yet expose a
	// per-instance setRemoteReference, so override granularity stops at the
	// fork-level macro for now.
	CopyRemoteReferences bool `json:"copyRemoteReferences,omitempty"`

	// Attributes are key/value labels set on the forked environment. Required
	// when the organization declares attributes at the environment scope (ABAC
	// gates `environment:create` on attribute-shaped policies). Both keys and
	// values must be strings. CLI flag `-a/--attributes` overrides this.
	Attributes map[string]string `json:"attributes,omitempty"`

	// EnvironmentDefaults pins specific resources as the env's defaults of their
	// type. Each entry must point at an existing resource.
	EnvironmentDefaults []EnvironmentDefaultEntry `json:"environmentDefaults,omitempty"`

	// Instances lists per-instance overrides. Listed instances without explicit
	// fields just inherit from the fork's seed.
	Instances map[string]InstanceOverride `json:"instances,omitempty"`
}

// EnvironmentDefaultEntry pins one resource as a default for the preview env.
// `resourceType` is documentation for the human reader; the CLI only needs
// `resourceId` for the API call.
type EnvironmentDefaultEntry struct {
	ResourceType string `json:"resourceType,omitempty"`
	ResourceID   string `json:"resourceId"`
}

// InstanceOverride captures the per-instance overrides for a preview env.
// Every field is optional; missing fields fall back to the value the fork
// seeded from the base environment.
//
// `version` accepts a semver constraint (e.g. `~2.0`, `1.2.3`, `latest`).
// Append `+dev` to pull from the development channel — e.g. `latest+dev` or
// `~2.0+dev`.
type InstanceOverride struct {
	Version string          `json:"version,omitempty"`
	Params  map[string]any  `json:"params,omitempty"`
	Secrets []PreviewSecret `json:"secrets,omitempty"`
}

// PreviewSecret is a single secret override on an instance.
type PreviewSecret struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PreviewOptions controls a single invocation of RunPreview.
type PreviewOptions struct {
	// ID is the local segment of the preview env identifier (e.g. "pr123").
	// Must match `^[a-z0-9]{1,20}$` — lowercase alphanumeric only, no dashes.
	ID string
	// Name is the human-readable name; defaults to ID.
	Name string
	// Description is the optional environment description.
	Description string
	// Attributes overrides `config.Attributes` when non-nil — useful for
	// piping CI metadata in via the CLI flag without rewriting the config.
	Attributes map[string]string
}

// PreviewAPI is the narrow SDK surface RunPreview needs. Tests supply a stub
// directly; production callers use [NewPreviewAPI] to bind a *massdriver.Client.
type PreviewAPI interface {
	Fork(ctx context.Context, parentID string, input environments.ForkInput) (*types.Environment, error)
	SetEnvironmentDefault(ctx context.Context, environmentID, resourceID string) error
	CopyInstance(ctx context.Context, sourceID, destinationID string, input instances.CopyInput) (*types.Instance, error)
	UpdateInstance(ctx context.Context, id string, input instances.UpdateInput) (*types.Instance, error)
	SetInstanceSecret(ctx context.Context, instanceID, name, value string) error
	DeployEnvironment(ctx context.Context, id string) (*types.Environment, error)
}

// NewPreviewAPI returns the production [PreviewAPI] backed by the SDK client.
func NewPreviewAPI(c *massdriver.Client) PreviewAPI { return sdkPreviewAPI{c: c} }

type sdkPreviewAPI struct{ c *massdriver.Client }

func (s sdkPreviewAPI) Fork(ctx context.Context, parentID string, input environments.ForkInput) (*types.Environment, error) {
	return s.c.Environments.Fork(ctx, parentID, input)
}

func (s sdkPreviewAPI) SetEnvironmentDefault(ctx context.Context, environmentID, resourceID string) error {
	_, err := s.c.Environments.SetDefault(ctx, environmentID, resourceID)
	return err
}

func (s sdkPreviewAPI) CopyInstance(ctx context.Context, sourceID, destinationID string, input instances.CopyInput) (*types.Instance, error) {
	return s.c.Instances.Copy(ctx, sourceID, destinationID, input)
}

func (s sdkPreviewAPI) UpdateInstance(ctx context.Context, id string, input instances.UpdateInput) (*types.Instance, error) {
	return s.c.Instances.Update(ctx, id, input)
}

func (s sdkPreviewAPI) SetInstanceSecret(ctx context.Context, instanceID, name, value string) error {
	_, err := s.c.Instances.SetSecret(ctx, instanceID, name, value)
	return err
}

func (s sdkPreviewAPI) DeployEnvironment(ctx context.Context, id string) (*types.Environment, error) {
	return s.c.Environments.Deploy(ctx, id)
}

// RunPreview converges a preview environment from `config`:
//
//  1. Fork the base environment.
//  2. Pin any environment defaults declared in the config.
//  3. Apply per-instance overrides (version, params, secrets).
//  4. Trigger a deploy of every instance in dependency order.
//
// Every step but (4) is idempotent — re-running the command against the same
// config converges the environment back to the declared state.
func RunPreview(ctx context.Context, api PreviewAPI, config *PreviewConfig, opts PreviewOptions) (*types.Environment, error) {
	if validateErr := validatePreviewConfig(config); validateErr != nil {
		return nil, validateErr
	}
	if opts.ID == "" {
		return nil, errors.New("preview environment ID is required")
	}

	parentID := fmt.Sprintf("%s-%s", config.Project, config.BaseEnvironment)
	previewID := fmt.Sprintf("%s-%s", config.Project, opts.ID)
	name := opts.Name
	if name == "" {
		name = opts.ID
	}

	attrs := config.Attributes
	if opts.Attributes != nil {
		attrs = opts.Attributes
	}

	fmt.Printf("⤴ Forking `%s` → `%s`\n", parentID, previewID)
	forkInput := environments.ForkInput{
		ID:                      opts.ID,
		Name:                    name,
		Description:             opts.Description,
		Attributes:              stringMapToAnyMap(attrs),
		CopyEnvironmentDefaults: config.CopyEnvironmentDefaults,
		CopySecrets:             config.CopySecrets,
		CopyRemoteReferences:    config.CopyRemoteReferences,
	}
	env, forkErr := api.Fork(ctx, parentID, forkInput)
	if forkErr != nil {
		return nil, fmt.Errorf("fork failed: %w", forkErr)
	}

	for _, ed := range config.EnvironmentDefaults {
		fmt.Printf("📌 Pinning environment default `%s`\n", ed.ResourceID)
		if edErr := api.SetEnvironmentDefault(ctx, previewID, ed.ResourceID); edErr != nil {
			return nil, fmt.Errorf("set environment default %s: %w", ed.ResourceID, edErr)
		}
	}

	for localID, override := range config.Instances {
		instanceID := fmt.Sprintf("%s-%s", previewID, localID)
		if applyErr := applyInstanceOverride(ctx, api, config, instanceID, localID, override); applyErr != nil {
			return nil, fmt.Errorf("instance %s: %w", instanceID, applyErr)
		}
	}

	fmt.Printf("🚀 Deploying `%s`\n", previewID)
	if _, deployErr := api.DeployEnvironment(ctx, previewID); deployErr != nil {
		return nil, fmt.Errorf("deploy failed: %w", deployErr)
	}

	return env, nil
}

// applyInstanceOverride applies the per-instance configuration in `override`
// to the preview env's instance. Order matters: params first (via copyInstance
// from the base env's matching instance, so it deep-merges over the parent's
// values), then version, then secrets.
func applyInstanceOverride(ctx context.Context, api PreviewAPI, config *PreviewConfig, instanceID, localID string, override InstanceOverride) error {
	if len(override.Params) > 0 {
		sourceID := fmt.Sprintf("%s-%s-%s", config.Project, config.BaseEnvironment, localID)
		fmt.Printf("📦 Configuring instance `%s`\n", instanceID)
		if _, copyErr := api.CopyInstance(ctx, sourceID, instanceID, instances.CopyInput{Overrides: override.Params}); copyErr != nil {
			return fmt.Errorf("copy params: %w", copyErr)
		}
	}

	if override.Version != "" {
		fmt.Printf("🏷  Pinning version on `%s`\n", instanceID)
		if _, updateErr := api.UpdateInstance(ctx, instanceID, instances.UpdateInput{Version: override.Version}); updateErr != nil {
			return fmt.Errorf("update version: %w", updateErr)
		}
	}

	for _, secret := range override.Secrets {
		fmt.Printf("🔐 Setting secret `%s` on `%s`\n", secret.Name, instanceID)
		if secretErr := api.SetInstanceSecret(ctx, instanceID, secret.Name, secret.Value); secretErr != nil {
			return fmt.Errorf("set secret %s: %w", secret.Name, secretErr)
		}
	}

	return nil
}

// LoadPreviewConfig reads and parses a preview config from `path`.
//
// `${VAR}` / `$VAR` references in the raw YAML are expanded from the
// process environment before parsing — so a config can read:
//
//	instances:
//	  chatsvc:
//	    params:
//	      host: chatty-pr-${GITHUB_PR}.example.com
//
// and pick up `GITHUB_PR` from the CI runner. Undefined variables expand to
// empty strings, matching `os.ExpandEnv`'s standard behavior.
func LoadPreviewConfig(path string) (*PreviewConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read preview config: %w", err)
	}
	expanded := os.ExpandEnv(string(data))
	cfg := &PreviewConfig{}
	if unmarshalErr := yaml.Unmarshal([]byte(expanded), cfg); unmarshalErr != nil {
		return nil, fmt.Errorf("parse preview config: %w", unmarshalErr)
	}
	return cfg, nil
}

// stringMapToAnyMap widens a string-valued map for SDK callers that expect
// `map[string]any` (matches the `:map` GraphQL type). Returns nil so the
// generated input struct keeps `attributes` absent when no overrides exist.
func stringMapToAnyMap(in map[string]string) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func validatePreviewConfig(config *PreviewConfig) error {
	if config == nil {
		return errors.New("preview config is required")
	}
	if config.Project == "" {
		return errors.New("preview config: `project` is required")
	}
	if config.BaseEnvironment == "" {
		return errors.New("preview config: `baseEnvironment` is required")
	}
	for i, ed := range config.EnvironmentDefaults {
		if ed.ResourceID == "" {
			return fmt.Errorf("preview config: environmentDefaults[%d]: `resourceId` is required", i)
		}
	}
	for localID, override := range config.Instances {
		for i, secret := range override.Secrets {
			if secret.Name == "" {
				return fmt.Errorf("preview config: instances.%s.secrets[%d]: `name` is required", localID, i)
			}
		}
	}
	return nil
}

package bundle

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/massdriver-cloud/mass/internal/files"
	"github.com/massdriver-cloud/mass/internal/prettylogs"
)

//go:embed schemas/metadata-schema.json
var embedFS embed.FS

var validSemverRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// MetadataSchema holds the parsed JSON schema for bundle metadata.
var MetadataSchema = parseMetadataSchema()

// ParamsFile and ConnsFile are the filenames used for auto-generated tfvars JSON files.
const (
	ParamsFile = "_params.auto.tfvars.json"
	ConnsFile  = "_connections.auto.tfvars.json"
)

// Step represents a single provisioner step within a bundle.
type Step struct {
	Path         string         `json:"path,omitempty" yaml:"path,omitempty" mapstructure:"path"`
	Provisioner  string         `json:"provisioner,omitempty" yaml:"provisioner,omitempty" mapstructure:"provisioner"`
	SkipOnDelete bool           `json:"skip_on_delete,omitempty" yaml:"skip_on_delete,omitempty" mapstructure:"skip_on_delete"`
	Config       map[string]any `json:"config,omitempty" yaml:"config,omitempty" mapstructure:"config"`
}

// AppSpec defines the application-specific configuration for environment variables, policies, and secrets.
type AppSpec struct {
	Envs     map[string]string `json:"envs" yaml:"envs" mapstructure:"envs"`
	Policies []string          `json:"policies" yaml:"policies" mapstructure:"policies"`
	Secrets  map[string]Secret `json:"secrets" yaml:"secrets" mapstructure:"secrets"`
}

// Secret describes a secret that the bundle expects to be injected at runtime.
type Secret struct {
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty" mapstructure:"required"`
	JSON        bool   `json:"json,omitempty" yaml:"json,omitempty" mapstructure:"json"`
	Title       string `json:"title,omitempty" yaml:"title,omitempty" mapstructure:"title"`
	Description string `json:"description,omitempty" yaml:"description,omitempty" mapstructure:"description"`
}

// Resource is one entry in a bundle's `resources` block — a resource the bundle
// produces.
type Resource struct {
	ResourceType string `json:"resource_type,omitempty" yaml:"resource_type,omitempty" mapstructure:"resource_type"`
	Required     *bool  `json:"required,omitempty" yaml:"required,omitempty" mapstructure:"required"`
}

// Dependency is one entry in a bundle's `dependencies` block — a resource the
// bundle depends on.
type Dependency struct {
	ResourceType string `json:"resource_type,omitempty" yaml:"resource_type,omitempty" mapstructure:"resource_type"`
	Required     *bool  `json:"required,omitempty" yaml:"required,omitempty" mapstructure:"required"`
}

// Bundle represents a Massdriver bundle definition parsed from massdriver.yaml.
type Bundle struct {
	Name        string         `json:"name,omitempty" yaml:"name,omitempty" mapstructure:"name"`
	Description string         `json:"description,omitempty" yaml:"description,omitempty" mapstructure:"description"`
	SourceURL   string         `json:"source_url,omitempty" yaml:"source_url,omitempty" mapstructure:"source_url"`
	Type        string         `json:"type,omitempty" yaml:"type,omitempty" mapstructure:"type"`
	Access      string         `json:"access,omitempty" yaml:"access,omitempty" mapstructure:"access"`
	Version     string         `json:"version,omitempty" yaml:"version,omitempty" mapstructure:"version"`
	Steps       []Step         `json:"steps,omitempty" yaml:"steps,omitempty" mapstructure:"steps"`
	Artifacts   map[string]any `json:"artifacts,omitempty" yaml:"artifacts,omitempty" mapstructure:"artifacts"`
	Params      map[string]any `json:"params,omitempty" yaml:"params,omitempty" mapstructure:"params"`
	Connections map[string]any `json:"connections,omitempty" yaml:"connections,omitempty" mapstructure:"connections"`
	UI          map[string]any `json:"ui,omitempty" yaml:"ui,omitempty" mapstructure:"ui"`
	AppSpec     *AppSpec       `json:"app,omitempty" yaml:"app,omitempty" mapstructure:"app"`

	// Resources and Dependencies are the current input terms. Artifacts and
	// Connections are their deprecated predecessors, accepted only at version
	// 0.0.0.
	Resources    map[string]Resource   `json:"resources,omitempty" yaml:"resources,omitempty" mapstructure:"resources"`
	Dependencies map[string]Dependency `json:"dependencies,omitempty" yaml:"dependencies,omitempty" mapstructure:"dependencies"`

	// dependencySchema is the canonical JSON-schema form of the bundle's
	// dependencies (from Dependencies or the legacy Connections block), hydrated
	// on demand and dereferenced in place by DereferenceSchemas.
	dependencySchema map[string]any
}

// Unmarshal reads and parses the massdriver.yaml file from the given directory into a Bundle.
func Unmarshal(readDirectory string) (*Bundle, error) {
	unmarshalledBundle := &Bundle{}
	if err := files.Read(filepath.Join(readDirectory, "massdriver.yaml"), unmarshalledBundle); err != nil {
		return nil, err
	}

	if unmarshalledBundle.Access != "" {
		fmt.Println(prettylogs.Orange("Warning: the 'access' field in massdriver.yaml is deprecated and should be removed."))
	}
	if unmarshalledBundle.Type != "" {
		fmt.Println(prettylogs.Orange("Warning: the 'type' field in massdriver.yaml is deprecated and should be removed."))
	}
	if unmarshalledBundle.Version == "" {
		fmt.Println(prettylogs.Orange("Warning: the 'version' field in massdriver.yaml is empty. This disables all versioning capabilities."))
		unmarshalledBundle.Version = "0.0.0"
	} else if !validSemverRegex.MatchString(unmarshalledBundle.Version) {
		return nil, fmt.Errorf("invalid version in massdriver.yaml: %s. Version must follow semantic versioning (MAJOR.MINOR.PATCH), e.g., 1.2.3", unmarshalledBundle.Version)
	}

	applyAppBlockDefaults(unmarshalledBundle)
	applyStepDefaults(unmarshalledBundle)

	if err := unmarshalledBundle.normalizeInputs(); err != nil {
		return nil, err
	}

	if transformationErr := ApplyTransformations(unmarshalledBundle.Params, paramsTransformations); transformationErr != nil {
		return nil, fmt.Errorf("failed to apply transformations to params: %w", transformationErr)
	}

	return unmarshalledBundle, nil
}

func applyAppBlockDefaults(b *Bundle) {
	if b.AppSpec != nil {
		if b.AppSpec.Envs == nil {
			b.AppSpec.Envs = map[string]string{}
		}
		if b.AppSpec.Policies == nil {
			b.AppSpec.Policies = []string{}
		}
		if b.AppSpec.Secrets == nil {
			b.AppSpec.Secrets = map[string]Secret{}
		}
	}
}

func applyStepDefaults(b *Bundle) {
	if len(b.Steps) == 0 {
		msg := fmt.Sprintf(`%s: No steps defined in massdriver.yaml, defaulting to Terraform provisioner. This will be deprecated in a future release. To avoid this warning, please add the following to massdriver.yaml:
steps:
  - path: src
    provisioner: terraform`, prettylogs.Orange("Warning"))
		fmt.Println(msg + "\n")
		b.Steps = append(b.Steps, Step{Path: "src", Provisioner: "terraform"})
	}
}

func parseMetadataSchema() map[string]any {
	metadataBytes, err := embedFS.ReadFile("schemas/metadata-schema.json")
	if err != nil {
		return nil
	}

	var metadata map[string]any
	err = json.Unmarshal(metadataBytes, &metadata)
	if err != nil {
		return nil
	}

	return metadata
}

// normalizeInputs validates the `resources`/`dependencies` blocks and enforces
// the rules around the legacy `artifacts`/`connections` terms: the two forms of a
// slot are mutually exclusive, and the legacy terms are only usable at version
// 0.0.0 (warn there, error at any real version). It does not write into the
// legacy fields — the dependency schema is hydrated separately.
func (b *Bundle) normalizeInputs() error {
	hasArtifacts := b.Artifacts != nil
	hasConnections := b.Connections != nil
	hasResources := b.Resources != nil
	hasDependencies := b.Dependencies != nil

	if hasConnections && hasDependencies {
		return errors.New("cannot set both 'connections' and 'dependencies'; use 'dependencies'")
	}
	if hasArtifacts && hasResources {
		return errors.New("cannot set both 'artifacts' and 'resources'; use 'resources'")
	}
	if hasConnections {
		if err := b.checkDeprecatedTerm("connections", "dependencies"); err != nil {
			return err
		}
	}
	if hasArtifacts {
		if err := b.checkDeprecatedTerm("artifacts", "resources"); err != nil {
			return err
		}
	}

	b.hydrateDependencySchema()
	return nil
}

// checkDeprecatedTerm enforces that a legacy term (artifacts/connections) is only
// usable at version 0.0.0: it warns at 0.0.0 and errors at any real version.
func (b *Bundle) checkDeprecatedTerm(oldTerm, newTerm string) error {
	if b.Version != "0.0.0" {
		return fmt.Errorf("the '%s' field is deprecated and doesn't support versioning; migrate to '%s' to publish version %s", oldTerm, newTerm, b.Version)
	}
	fmt.Println(prettylogs.Orange(fmt.Sprintf("Warning: the '%s' field is deprecated; migrate to '%s'. The legacy term does not support versioned resource types", oldTerm, newTerm)))
	return nil
}

// hydrateDependencySchema builds dependencySchema — the canonical JSON-schema map
// ({properties: {name: {$ref}}, required: [...]}) that downstream code (schema
// dereferencing, provisioner input generation, lint) reads for dependencies. It
// is sourced from `dependencies` (new) or the legacy `connections` block.
func (b *Bundle) hydrateDependencySchema() {
	switch {
	case len(b.Dependencies) > 0:
		b.dependencySchema = dependenciesToSchema(b.Dependencies)
	case b.Connections != nil:
		if _, ok := b.Connections["properties"].(map[string]any); !ok {
			b.Connections["properties"] = map[string]any{}
		}
		b.dependencySchema = b.Connections
	default:
		b.dependencySchema = map[string]any{"properties": map[string]any{}}
	}
}

// dependenciesToSchema converts a `dependencies` map into the canonical JSON
// schema. Per-entry validation (resource_type/required presence) is handled by
// bundle schema validation, before dereferencing.
func dependenciesToSchema(deps map[string]Dependency) map[string]any {
	properties := map[string]any{}
	required := []any{}
	for name, dep := range deps {
		properties[name] = map[string]any{"$ref": dep.ResourceType}
		if dep.Required != nil && *dep.Required {
			required = append(required, name)
		}
	}
	return map[string]any{"properties": properties, "required": required}
}

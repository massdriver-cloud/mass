// Package resourcetype holds the testable logic behind the `mass resource-type`
// commands. The cobra wiring lives in the top-level cmd package; generalized,
// reusable resource-type logic lives in internal/resourcetype.
package resourcetype

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/jsonschema"
	"github.com/massdriver-cloud/mass/internal/oci"
	"github.com/massdriver-cloud/mass/internal/prettylogs"
	"github.com/massdriver-cloud/mass/internal/resourcetype"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"oras.land/oras-go/v2/content/memory"
)

// allowedFiles is the exact set of top-level files that may be packaged into a
// resource type artifact. readme/changelog are listed in both their
// conventional uppercase and lowercase forms; everything else at the top level
// is silently skipped.
var allowedFiles = []string{
	"massdriver.yaml",
	"README.md",
	"readme.md",
	"CHANGELOG.md",
	"changelog.md",
	"icon.svg",
	"icon.png",
	"icon.jpg",
	"icon.jpeg",
}

// referencedPaths returns the raw instruction and export template file
// references declared in a massdriver.yaml, in declaration order.
func referencedPaths(config *resourcetype.MassdriverYAML) []string {
	var refs []string
	if config.UI != nil {
		for _, inst := range config.UI.Instructions {
			refs = append(refs, inst.Path)
		}
	}
	for _, exp := range config.Exports {
		refs = append(refs, exp.TemplatePath)
	}
	return refs
}

// packageKeep builds the keep predicate used when packaging a resource type.
// It admits the allowlisted top-level files plus the exact instruction and
// export template files the massdriver.yaml references (wherever they live in
// the directory tree), and silently skips everything else.
func packageKeep(config *resourcetype.MassdriverYAML) func(relPath string) bool {
	referenced := map[string]bool{}
	for _, p := range referencedPaths(config) {
		if norm := normalizeRel(p); norm != "" {
			referenced[norm] = true
		}
	}

	return func(relPath string) bool {
		return referenced[relPath] || slices.Contains(allowedFiles, relPath)
	}
}

// validateReferencedFiles ensures every instruction/export file the
// massdriver.yaml references resolves to a real file inside srcDir. References
// that are absolute, escape the directory, or don't exist would be dropped by
// the packager and produce a silently incomplete artifact, so they're rejected
// up front.
func validateReferencedFiles(config *resourcetype.MassdriverYAML, srcDir string) error {
	for _, ref := range referencedPaths(config) {
		if ref == "" {
			continue
		}
		norm := normalizeRel(ref)
		if norm == "" {
			return fmt.Errorf("referenced file %q must live inside the resource type directory (absolute paths and paths outside the directory can't be packaged)", ref)
		}
		info, statErr := os.Stat(filepath.Join(srcDir, norm))
		if statErr != nil {
			return fmt.Errorf("referenced file %q was not found in the resource type directory: %w", ref, statErr)
		}
		if info.IsDir() {
			return fmt.Errorf("referenced file %q is a directory, not a file", ref)
		}
	}
	return nil
}

// normalizeRel converts a massdriver.yaml file reference (relative to the
// massdriver.yaml, e.g. "./instructions/cli.md") into the slash-separated,
// cleaned form the packager's keep predicate receives. Empty and non-local
// (absolute or parent-escaping) references return "" since they can't match a
// file walked under the resource type directory.
func normalizeRel(p string) string {
	if p == "" {
		return ""
	}
	cleaned := filepath.ToSlash(filepath.Clean(p))
	if cleaned == "." || filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "../") {
		return ""
	}
	return cleaned
}

// RunPublish validates a resource type located at path and publishes it. path
// may be a directory containing a massdriver.yaml, the massdriver.yaml itself,
// or — via the deprecated legacy path — a raw JSON/YAML schema file. It returns
// the resource type name and the published version.
func RunPublish(ctx context.Context, mdClient *massdriver.Client, path string) (string, string, error) {
	target, resolveErr := resolvePublishPath(path)
	if resolveErr != nil {
		return "", "", resolveErr
	}
	if target.legacy {
		return publishLegacySchema(ctx, mdClient, target.path)
	}

	mdYamlPath, srcDir := target.path, target.srcDir

	config, configErr := resourcetype.ReadConfig(mdYamlPath)
	if configErr != nil {
		return "", "", fmt.Errorf("failed to read massdriver.yaml: %w", configErr)
	}
	if config.Name == "" {
		return "", "", fmt.Errorf("name is required in %s", mdYamlPath)
	}
	if config.Version == "" {
		fmt.Println(prettylogs.Orange("Warning: the 'version' field in massdriver.yaml is empty. This disables all versioning capabilities."))
		config.Version = "0.0.0"
	}

	// Referenced instruction/export files must live inside the packaged
	// directory, otherwise the artifact would ship incomplete.
	if refErr := validateReferencedFiles(config, srcDir); refErr != nil {
		return "", "", refErr
	}

	// Fail fast on a duplicate version before the network-heavy schema
	// dereference and validation.
	if versionErr := checkDuplicateVersion(ctx, mdClient, config.Name, config.Version); versionErr != nil {
		return "", "", versionErr
	}

	if validateErr := validateSchema(ctx, mdClient, mdYamlPath); validateErr != nil {
		return "", "", validateErr
	}

	repo, repoErr := mdClient.OciRepos.Target(config.Name)
	if repoErr != nil {
		return "", "", fmt.Errorf("getting repository: %w", repoErr)
	}

	publisher := &oci.Publisher{
		Store: memory.New(),
		Repo:  repo,
	}

	if _, packageErr := publisher.Package(ctx, srcDir, config.Version, resourcetype.ArtifactType, packageKeep(config)); packageErr != nil {
		return "", "", fmt.Errorf("packaging resource type: %w", packageErr)
	}

	if publishErr := publisher.Publish(ctx, config.Version); publishErr != nil {
		return "", "", fmt.Errorf("publishing resource type: %w", publishErr)
	}

	return config.Name, config.Version, nil
}

// publishTarget is the resolved shape of a publish argument: either a
// massdriver.yaml plus the directory to package (the OCI flow), or a raw
// JSON/YAML schema file (the deprecated legacy flow).
type publishTarget struct {
	// path is the massdriver.yaml, or the raw schema file when legacy is set.
	path string
	// srcDir is the directory packaged into the OCI artifact. Unused when
	// legacy is set — the legacy mutation publishes a schema document, not a
	// directory.
	srcDir string
	legacy bool
}

// resolvePublishPath resolves the publish target. A directory or massdriver.yaml
// takes the OCI flow; a bare .json/.yaml/.yml schema file takes the deprecated
// legacy flow.
func resolvePublishPath(path string) (publishTarget, error) {
	info, statErr := os.Stat(path)
	if statErr != nil {
		return publishTarget{}, fmt.Errorf("failed to read resource type path: %w", statErr)
	}

	if info.IsDir() {
		md := filepath.Join(path, "massdriver.yaml")
		if _, mdErr := os.Stat(md); mdErr != nil {
			return publishTarget{}, fmt.Errorf("no massdriver.yaml found in %s", path)
		}
		return publishTarget{path: md, srcDir: path}, nil
	}

	if filepath.Base(path) == "massdriver.yaml" {
		return publishTarget{path: path, srcDir: filepath.Dir(path)}, nil
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".yaml", ".yml":
		return publishTarget{path: path, legacy: true}, nil
	default:
		return publishTarget{}, fmt.Errorf("unsupported resource type path: %s (expected a directory, a massdriver.yaml, or a JSON schema file)", path)
	}
}

// publishLegacySchema publishes a raw JSON/YAML schema document through the
// deprecated `publishResourceType` mutation. The schema has no version of its
// own, so the API stores it as the resource type's unversioned 0.0.0 document —
// which is why this flow can't participate in resource type versioning and is
// on its way out.
func publishLegacySchema(ctx context.Context, mdClient *massdriver.Client, path string) (string, string, error) {
	// Warn before any work so the notice lands whether or not the publish
	// itself succeeds.
	warnLegacySchema(path)

	rt, readErr := resourcetype.Read(ctx, mdClient, path)
	if readErr != nil {
		return "", "", fmt.Errorf("failed to read resource type: %w", readErr)
	}

	if validateErr := validateBuiltSchema(mdClient, rt); validateErr != nil {
		return "", "", validateErr
	}

	published, publishErr := api.PublishResourceType(ctx, mdClient, api.PublishResourceTypeInput{Schema: rt})
	if publishErr != nil {
		return "", "", publishErr
	}

	version := published.Version
	if version == "" {
		version = legacySchemaVersion
	}
	return published.Name, version, nil
}

// legacySchemaVersion is the unversioned document the legacy mutation writes to.
// Used only as a display fallback if the API omits the version in its response.
const legacySchemaVersion = "0.0.0"

// warnLegacySchema tells the user their raw JSON schema is on a deprecated,
// unversioned path and points them at `resource-type convert`.
// Printed as separate lines rather than one multi-line string: lipgloss pads
// every line of a styled block to the width of its longest line, which leaves
// ragged trailing whitespace once a long file path is interpolated in.
func warnLegacySchema(path string) {
	fmt.Println(prettylogs.Orange("Warning: this resource type is a raw JSON schema. That format is deprecated, does not support versioning, and will be removed in a future release. Migrate it to the massdriver.yaml format, which supports versioning, by running:"))
	fmt.Println(prettylogs.Orange(fmt.Sprintf("    mass resource-type convert %s", path)))
}

// validateSchema builds and dereferences the resource type, then validates it
// against the resource type schema and the JSON Schema meta-schema.
func validateSchema(ctx context.Context, mdClient *massdriver.Client, mdYamlPath string) error {
	rt, readErr := resourcetype.Read(ctx, mdClient, mdYamlPath)
	if readErr != nil {
		return fmt.Errorf("failed to read resource type: %w", readErr)
	}
	return validateBuiltSchema(mdClient, rt)
}

// validateBuiltSchema validates an already read-and-dereferenced resource type
// against the resource type schema and the JSON Schema meta-schema.
func validateBuiltSchema(mdClient *massdriver.Client, rt map[string]any) error {
	cfg := mdClient.Config()
	rtSchemaURL, err := url.JoinPath(cfg.URL, "json-schemas", "resource-type.json")
	if err != nil {
		return fmt.Errorf("failed to construct resource type schema URL: %w", err)
	}
	if validateErr := validateResourceType(rt, rtSchemaURL); validateErr != nil {
		return fmt.Errorf("failed to validate resource type schema: %w", validateErr)
	}

	metaSchemaURL, err := url.JoinPath(cfg.URL, "json-schemas", "draft-7.json")
	if err != nil {
		return fmt.Errorf("failed to construct meta schema URL: %w", err)
	}
	if validateErr := validateResourceType(rt, metaSchemaURL); validateErr != nil {
		return fmt.Errorf("failed to validate resource type against meta schema: %w", validateErr)
	}

	return nil
}

// checkDuplicateVersion fails locally if version has already been published,
// matching the immutability the API enforces.
func checkDuplicateVersion(ctx context.Context, mdClient *massdriver.Client, name, version string) error {
	// 0.0.0 is the unversioned/dev tag — always republishable, matching bundles.
	if version == "0.0.0" {
		return nil
	}
	repo, err := mdClient.OciRepos.Get(ctx, name)
	if err != nil {
		return fmt.Errorf("fetching OCI repo: %w", err)
	}
	for _, t := range repo.Tags {
		if t.Tag == version {
			return fmt.Errorf("version %s already exists for resource type %s", version, name)
		}
	}
	return nil
}

func validateResourceType(rt map[string]any, schemaURL string) error {
	sch, loadErr := jsonschema.LoadSchemaFromURL(schemaURL)
	if loadErr != nil {
		return loadErr
	}
	return jsonschema.ValidateGo(sch, rt)
}

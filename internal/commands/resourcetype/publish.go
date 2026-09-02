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
	"github.com/massdriver-cloud/mass/internal/commands/repository"
	"github.com/massdriver-cloud/mass/internal/jsonschema"
	"github.com/massdriver-cloud/mass/internal/oci"
	"github.com/massdriver-cloud/mass/internal/prettylogs"
	"github.com/massdriver-cloud/mass/internal/resourcetype"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"oras.land/oras-go/v2/content/memory"
)

// Everything else at the top level is silently skipped.
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

// References the packager would drop are rejected up front: they'd ship a
// silently incomplete artifact.
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

// normalizeRel renders a massdriver.yaml file reference in the form the keep
// predicate receives. Non-local references return "" — nothing can match them.
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

// RunPublish publishes the resource type at path — a directory, a
// massdriver.yaml, or a raw schema file via the deprecated legacy path.
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

	if refErr := validateReferencedFiles(config, srcDir); refErr != nil {
		return "", "", refErr
	}

	// Before the network-heavy dereference and validation.
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

type publishTarget struct {
	// The massdriver.yaml, or the raw schema file when legacy is set.
	path string
	// Directory packaged into the OCI artifact. Unused when legacy is set.
	srcDir string
	legacy bool
}

// A directory or massdriver.yaml takes the OCI flow; a bare schema file takes
// the deprecated legacy flow.
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

// The schema has no version of its own, so the API stores it as the
// unversioned 0.0.0 document — hence no versioning support.
func publishLegacySchema(ctx context.Context, mdClient *massdriver.Client, path string) (string, string, error) {
	// Before any work, so it lands even if the publish fails.
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

	// published.Name is the human label; the identifier is the ID's prefix.
	name := published.Name
	if identifier, _, found := strings.Cut(published.ID, "@"); found && identifier != "" {
		name = identifier
	}
	return name, version, nil
}

const legacySchemaVersion = "0.0.0"

func warnLegacySchema(path string) {
	fmt.Println(prettylogs.Orange("Warning: this resource type is a raw JSON schema. That format is deprecated, does not support versioning, and will be removed in a future release. Migrate it to the massdriver.yaml format, which supports versioning, by running:"))
	fmt.Println(prettylogs.Orange(fmt.Sprintf("    mass resource-type convert %s", path)))
}

func validateSchema(ctx context.Context, mdClient *massdriver.Client, mdYamlPath string) error {
	rt, readErr := resourcetype.Read(ctx, mdClient, mdYamlPath)
	if readErr != nil {
		return fmt.Errorf("failed to read resource type: %w", readErr)
	}
	return validateBuiltSchema(mdClient, rt)
}

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

// checkDuplicateVersion mirrors the immutability the API enforces, failing
// before the network-heavy packaging work.
func checkDuplicateVersion(ctx context.Context, mdClient *massdriver.Client, name, version string) error {
	// 0.0.0 is the unversioned/dev tag — republishable, matching bundles, though
	// the API refuses it once other versions exist.
	if version == "0.0.0" {
		return nil
	}
	repo, err := mdClient.OciRepos.Get(ctx, name)
	if err != nil {
		return repository.NotFoundHint(err, "resource-type", name)
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

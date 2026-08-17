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

	"github.com/massdriver-cloud/mass/internal/jsonschema"
	"github.com/massdriver-cloud/mass/internal/oci"
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

// RunPublish validates a resource type located at path and pushes it to its OCI
// repository. path may be a directory containing a massdriver.yaml, or the
// massdriver.yaml itself. It returns the resource type name and the published
// version.
func RunPublish(ctx context.Context, mdClient *massdriver.Client, path string) (string, string, error) {
	mdYamlPath, srcDir, resolveErr := resolvePublishPath(path)
	if resolveErr != nil {
		return "", "", resolveErr
	}

	config, configErr := resourcetype.ReadConfig(mdYamlPath)
	if configErr != nil {
		return "", "", fmt.Errorf("failed to read massdriver.yaml: %w", configErr)
	}
	if config.Name == "" {
		return "", "", fmt.Errorf("name is required in %s", mdYamlPath)
	}
	if config.Version == "" {
		return "", "", fmt.Errorf("version is required in %s", mdYamlPath)
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

// resolvePublishPath resolves the publish target into the massdriver.yaml path
// and its containing directory, rejecting raw JSON schema files with a pointer
// to the convert command.
func resolvePublishPath(path string) (mdYamlPath string, srcDir string, err error) {
	info, statErr := os.Stat(path)
	if statErr != nil {
		return "", "", fmt.Errorf("failed to read resource type path: %w", statErr)
	}

	if info.IsDir() {
		md := filepath.Join(path, "massdriver.yaml")
		if _, mdErr := os.Stat(md); mdErr != nil {
			return "", "", fmt.Errorf("no massdriver.yaml found in %s", path)
		}
		return md, path, nil
	}

	if filepath.Base(path) == "massdriver.yaml" {
		return path, filepath.Dir(path), nil
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".yaml", ".yml":
		return "", "", fmt.Errorf("publishing a raw JSON schema is no longer supported; run `mass resource-type convert %s` to migrate it to a massdriver.yaml", path)
	default:
		return "", "", fmt.Errorf("unsupported resource type path: %s (expected a directory or massdriver.yaml)", path)
	}
}

// validateSchema builds and dereferences the resource type, then validates it
// against the resource type schema and the JSON Schema meta-schema.
func validateSchema(ctx context.Context, mdClient *massdriver.Client, mdYamlPath string) error {
	rt, readErr := resourcetype.Read(ctx, mdClient, mdYamlPath)
	if readErr != nil {
		return fmt.Errorf("failed to read resource type: %w", readErr)
	}

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

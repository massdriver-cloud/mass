// Package repository holds the testable logic behind the `mass repository`
// CLI commands. The cobra wiring lives in the top-level cmd package; anything
// worth a unit test belongs here.
package repository

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/ocirepos"
)

// artifactTypeAliases maps the user-facing --type flag values to the SDK's
// typed artifact-type enum.
var artifactTypeAliases = map[string]ocirepos.ArtifactType{
	"bundle":        ocirepos.ArtifactTypeBundle,
	"resource-type": ocirepos.ArtifactTypeResourceType,
}

// artifactTypeLabels is the reverse lookup for table/output rendering — turns
// the SDK's typed enum back into the friendly name the user typed.
var artifactTypeLabels = map[ocirepos.ArtifactType]string{
	ocirepos.ArtifactTypeBundle:       "bundle",
	ocirepos.ArtifactTypeResourceType: "resource-type",
}

// ResolveArtifactType converts a user-facing alias (e.g. "bundle",
// "resource-type") into the SDK's typed enum. Matching is case-insensitive;
// underscores match hyphens so the SDK's own "RESOURCE_TYPE" resolves too.
func ResolveArtifactType(s string) (ocirepos.ArtifactType, error) {
	normalized := strings.ReplaceAll(strings.ToLower(s), "_", "-")
	if at, ok := artifactTypeAliases[normalized]; ok {
		return at, nil
	}
	return "", fmt.Errorf("unknown artifact type %q (valid: %s)", s, strings.Join(ValidArtifactTypes(), ", "))
}

// ArtifactTypeLabel renders an enum value back to its friendly alias, falling
// back to the raw enum string for values without a known label.
func ArtifactTypeLabel(at ocirepos.ArtifactType) string {
	if label, ok := artifactTypeLabels[at]; ok {
		return label
	}
	return string(at)
}

// ValidArtifactTypes returns the user-facing artifact-type aliases in a
// deterministic (sorted) order, for flag help and error messages.
func ValidArtifactTypes() []string {
	valid := make([]string, 0, len(artifactTypeAliases))
	for k := range artifactTypeAliases {
		valid = append(valid, k)
	}
	sort.Strings(valid)
	return valid
}

// NotFoundHint replaces a not-found error with one naming the create command.
func NotFoundHint(err error, artifactType, name string) error {
	if !errors.Is(err, gql.ErrNotFound) {
		return err
	}
	return fmt.Errorf("%s %q does not exist. Create it with: mass %s create %s", artifactType, name, artifactType, name)
}

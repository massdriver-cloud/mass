// Package bundle provides command implementations for bundle operations.
package bundle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/massdriver-cloud/mass/internal/api"
	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/prettylogs"
	"github.com/massdriver-cloud/mass/internal/resourcetype"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
)

// RunBuild validates the bundle, warns about dependency cardinality changes,
// then builds it at buildPath.
func RunBuild(ctx context.Context, buildPath string, b *bundle.Bundle, mdClient *massdriver.Client, out io.Writer) error {
	if err := ValidateSchema(b, mdClient.Config().URL); err != nil {
		return err
	}
	WarnCardinalityChanges(ctx, NewPublishedBundleAPI(mdClient), b, out)
	return b.Build(buildPath, resourcetype.NewMassdriverResolver(mdClient))
}

// ValidateSchema must run before dereferencing, which assumes a schema-valid
// bundle. A fetch failure halts the build — the API owns the bundle format.
func ValidateSchema(b *bundle.Bundle, serverURL string) error {
	result := b.LintSchema(serverURL)
	if !result.HasErrors() {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("bundle failed schema validation:")
	for _, issue := range result.Errors() {
		fmt.Fprintf(&sb, "\n  - %s", issue.Message)
	}
	return errors.New(sb.String())
}

// PublishedBundleAPI reads the latest published version of a bundle.
type PublishedBundleAPI interface {
	// PublishedCardinalities returns the latest published version of the named
	// bundle and each dependency's cardinality ("one" or "many"), keyed by name.
	PublishedCardinalities(ctx context.Context, name string) (string, map[string]string, error)
}

// NewPublishedBundleAPI returns the production [PublishedBundleAPI].
func NewPublishedBundleAPI(c *massdriver.Client) PublishedBundleAPI { return publishedBundleAPI{c: c} }

type publishedBundleAPI struct{ c *massdriver.Client }

func (a publishedBundleAPI) PublishedCardinalities(ctx context.Context, name string) (string, map[string]string, error) {
	published, err := api.GetBundleDependencyCardinalities(ctx, a.c, name)
	if err != nil {
		return "", nil, err
	}
	cardinalities := make(map[string]string, len(published.Cardinalities))
	for field, cardinality := range published.Cardinalities {
		cardinalities[field] = strings.ToLower(cardinality)
	}
	return published.Version, cardinalities, nil
}

// WarnCardinalityChanges writes a warning for each dependency whose
// cardinality differs from the latest published version of the bundle.
// The check is advisory: when the published version cannot be read, for any
// reason, it writes nothing.
func WarnCardinalityChanges(ctx context.Context, published PublishedBundleAPI, b *bundle.Bundle, out io.Writer) {
	if len(b.Dependencies) == 0 {
		return
	}
	version, cardinalities, err := published.PublishedCardinalities(ctx, b.Name)
	if err != nil {
		return
	}

	for _, name := range slices.Sorted(maps.Keys(b.Dependencies)) {
		was, found := cardinalities[name]
		if !found {
			continue
		}
		now := bundle.CardinalityOne
		if b.Dependencies[name].IsMany() {
			now = bundle.CardinalityMany
		}
		if was == now {
			continue
		}
		effect := "The variable becomes a map; the existing line becomes a set of one."
		if now == bundle.CardinalityOne {
			effect = "The variable becomes an object; on upgrade the field's connections are removed and the user connects one source."
		}
		fmt.Fprintln(out, prettylogs.Orange(fmt.Sprintf(
			"Warning: dependency %q changes cardinality from %s to %s since %s@%s. %s",
			name, was, now, b.Name, version, effect)))
	}
}

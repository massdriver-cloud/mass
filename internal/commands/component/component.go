// Package component provides command implementations for managing components in a project's blueprint.
package component

import (
	"fmt"
	"strings"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// SplitComponentID splits a full component ID (e.g., "ecomm-db") into its project and short ID.
// Both segments are lowercase-alphanumeric per the API contract, so splitting on the first
// hyphen is unambiguous.
func SplitComponentID(fullID string) (projectID, shortID string, err error) {
	idx := strings.Index(fullID, "-")
	if idx <= 0 || idx == len(fullID)-1 {
		return "", "", fmt.Errorf("invalid component ID %q: expected <project-id>-<component-id>", fullID)
	}
	return fullID[:idx], fullID[idx+1:], nil
}

// ParseComponentField parses "<componentID>.<field>" into its parts.
func ParseComponentField(arg string) (componentID, field string, err error) {
	idx := strings.Index(arg, ".")
	if idx <= 0 || idx == len(arg)-1 {
		return "", "", fmt.Errorf("invalid argument %q: expected <component-id>.<field>", arg)
	}
	return arg[:idx], arg[idx+1:], nil
}

// FindLink returns the link whose from/to component+field tuple matches all
// four arguments. The platform no longer exposes a links-list query scoped by
// component, so callers pass the full project link set and FindLink filters
// in-memory.
func FindLink(links []types.Link, fromComponentID, fromField, toComponentID, toField string) (*types.Link, error) {
	for i := range links {
		l := &links[i]
		if l.FromComponent == nil || l.ToComponent == nil {
			continue
		}
		if l.FromComponent.ID == fromComponentID &&
			l.FromField == fromField &&
			l.ToComponent.ID == toComponentID &&
			l.ToField == toField {
			return l, nil
		}
	}
	return nil, fmt.Errorf("no link found from %s.%s to %s.%s", fromComponentID, fromField, toComponentID, toField)
}

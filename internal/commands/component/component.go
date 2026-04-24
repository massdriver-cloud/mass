// Package component provides command implementations for managing components in a project's blueprint.
package component

import (
	"fmt"
	"strings"

	api "github.com/massdriver-cloud/mass/internal/api/v1"
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

// FindLink returns the link whose FromField/ToField match, or an error if none exists.
func FindLink(links []api.Link, fromField, toField string) (*api.Link, error) {
	for i := range links {
		if links[i].FromField == fromField && links[i].ToField == toField {
			return &links[i], nil
		}
	}
	return nil, fmt.Errorf("no link found with fromField=%q toField=%q", fromField, toField)
}

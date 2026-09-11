package grants

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

const wildcardValue = "*"

// Repeating a key unions its values; only allFlag yields nil, the org-wide
// wildcard.
func ResolveConditions(pairs []string, file string, all bool, allFlag string) (types.PolicyConditions, error) {
	supplied := 0
	for _, given := range []bool{len(pairs) > 0, file != "", all} {
		if given {
			supplied++
		}
	}

	switch {
	case supplied == 0:
		return nil, fmt.Errorf("specify recipient conditions with --condition or --conditions-file, or share with your whole organization using --%s", allFlag)
	case supplied > 1:
		return nil, fmt.Errorf("--condition, --conditions-file, and --%s are mutually exclusive", allFlag)
	case all:
		//nolint:nilnil // nil conditions is the org-wide wildcard, not a missing value
		return nil, nil
	case file != "":
		return ParseConditionsFile(file, allFlag)
	default:
		return ParseConditions(pairs)
	}
}

func ParseConditions(pairs []string) (types.PolicyConditions, error) {
	if len(pairs) == 0 {
		return nil, errors.New("no conditions given")
	}

	conditions := types.PolicyConditions{}
	wildcards := map[string]bool{}

	for _, pair := range pairs {
		key, value, found := strings.Cut(pair, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch {
		case !found:
			return nil, fmt.Errorf("condition %q must be formatted as key=value", pair)
		case key == "":
			return nil, fmt.Errorf("condition %q has an empty attribute name", pair)
		case value == "":
			return nil, fmt.Errorf("condition %q has an empty value; write %s=%s to accept any value", pair, key, wildcardValue)
		}

		if value == wildcardValue {
			if len(conditions[key]) > 0 {
				return nil, conflictError(key)
			}
			wildcards[key] = true
			conditions[key] = nil
			continue
		}
		if wildcards[key] {
			return nil, conflictError(key)
		}
		if !slices.Contains(conditions[key], value) {
			conditions[key] = append(conditions[key], value)
		}
	}

	return conditions, nil
}

func ParseConditionsFile(path, allFlag string) (types.PolicyConditions, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading conditions file: %w", err)
	}

	var conditions types.PolicyConditions
	if unmarshalErr := json.Unmarshal(data, &conditions); unmarshalErr != nil {
		return nil, fmt.Errorf("parsing conditions file %s: %w", path, unmarshalErr)
	}

	// `null` and `{}` decode to the org-wide wildcard, routing around allFlag.
	if len(conditions) == 0 {
		return nil, fmt.Errorf("conditions file %s describes no conditions; share with your whole organization using --%s", path, allFlag)
	}
	for key, values := range conditions {
		if strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("conditions file %s has an empty attribute name", path)
		}
		for i, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				return nil, fmt.Errorf("conditions file %s has an empty value for %q; use %q to accept any value", path, key, wildcardValue)
			}
			values[i] = trimmed
		}
	}

	return conditions, nil
}

func FormatConditions(conditions types.PolicyConditions) string {
	if len(conditions) == 0 {
		return "everyone"
	}

	keys := make([]string, 0, len(conditions))
	for key := range conditions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		if len(conditions[key]) == 0 {
			parts = append(parts, key+"="+wildcardValue)
			continue
		}
		parts = append(parts, key+"="+strings.Join(conditions[key], "|"))
	}
	return strings.Join(parts, ", ")
}

func conflictError(key string) error {
	return fmt.Errorf("condition %q mixes %s=%s with specific values; use one or the other", key, key, wildcardValue)
}

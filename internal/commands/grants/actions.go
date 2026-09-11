package grants

import (
	"fmt"
	"slices"
	"strings"
)

// The read path returns actions that cannot be granted (repo:view, repo:push,
// resource:view), so these gate --action only.
const (
	ActionRepoPull       = "repo:pull"
	ActionResourceExport = "resource:export"
)

var (
	grantableRepoActions     = []string{ActionRepoPull}
	grantableResourceActions = []string{ActionResourceExport}
)

func ResolveRepoAction(action string) (string, error) {
	return resolveAction(action, ActionRepoPull, grantableRepoActions)
}

func ResolveResourceAction(action string) (string, error) {
	return resolveAction(action, ActionResourceExport, grantableResourceActions)
}

func resolveAction(action, fallback string, grantable []string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(action))
	if normalized == "" {
		return fallback, nil
	}
	if slices.Contains(grantable, normalized) {
		return normalized, nil
	}
	return "", fmt.Errorf("unknown action %q (valid: %s)", action, strings.Join(grantable, ", "))
}

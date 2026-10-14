package bundle

import (
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/massdriver-cloud/mass/internal/files"
)

// ExtendsKey is the massdriver.yaml key naming the parent bundle.
const ExtendsKey = "extends"

// DocFiles are the top-level files a child carries over its parent's copy.
var DocFiles = []string{"README.md", "readme.md", "CHANGELOG.md", "icon.svg", "operator.md", "operator.mdx"}

var childMapKeys = []string{"app", "dependencies", "resources", "steps"}

var childStepKeys = []string{"path", "resources"}

var resourceFieldRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var legacyInputKeys = []string{"connections", "artifacts"}

var envReferenceRegex = regexp.MustCompile(`\.(params|dependencies|connections)\.([A-Za-z0-9_-]+)`)

// ParentRef identifies the exact parent a child was merged with. Its map form
// replaces the child's `extends` constraint in the merged massdriver.yaml.
type ParentRef struct {
	Name       string
	Constraint string
	Version    string
	Digest     string
}

func (r ParentRef) toMap() map[string]any {
	return map[string]any{
		"name":       r.Name,
		"constraint": r.Constraint,
		"version":    r.Version,
		"digest":     r.Digest,
	}
}

// ReadSpec reads massdriver.yaml from dir as an untyped document.
func ReadSpec(dir string) (map[string]any, error) {
	spec := map[string]any{}
	if err := files.Read(filepath.Join(dir, "massdriver.yaml"), &spec); err != nil {
		return nil, err
	}
	return spec, nil
}

// ParseExtends splits a child's `extends` value into a parent name and
// version constraint. A bare name tracks `latest`.
func ParseExtends(spec map[string]any) (name, constraint string, found bool, err error) {
	raw, ok := spec[ExtendsKey]
	if !ok {
		return "", "", false, nil
	}
	ref, isString := raw.(string)
	if !isString || ref == "" {
		return "", "", true, fmt.Errorf("'%s' must be a string like <bundle>@<constraint>, e.g. lambda@~1.2", ExtendsKey)
	}
	name, constraint, hasConstraint := strings.Cut(ref, "@")
	if !hasConstraint {
		constraint = "latest"
	}
	if strings.Contains(name, "/") {
		return "", "", true, fmt.Errorf("'%s: %s' references another organization; a parent must be in your organization", ExtendsKey, ref)
	}
	if name == "" || constraint == "" {
		return "", "", true, fmt.Errorf("'%s: %s' must be <bundle>@<constraint>, e.g. lambda@~1.2", ExtendsKey, ref)
	}
	return name, constraint, true, nil
}

// StepResource is a jq template the child binds to one of the parent's steps.
// The provisioner renders it after that step applies and publishes the result
// as the resource Field.
type StepResource struct {
	Step  string
	Field string
	Query string
}

// Merge is the result of overlaying a child onto its parent. Warnings are
// non-fatal.
type Merge struct {
	Spec          map[string]any
	StepResources []StepResource
	Warnings      []string
}

// MergeExtends overlays a child massdriver.yaml onto its parent's. The parent
// owns everything that shapes the IaC; the child supplies identity, extra
// dependencies and resources, and app wiring.
func MergeExtends(parent, child map[string]any, ref ParentRef) (Merge, error) {
	if err := validateChildKeys(child); err != nil {
		return Merge{}, err
	}
	if _, ok := child["version"].(string); !ok {
		return Merge{}, errors.New("a bundle that extends another must set its own 'version'")
	}
	for _, key := range legacyInputKeys {
		if _, ok := parent[key]; ok {
			return Merge{}, fmt.Errorf("parent %s@%s uses the deprecated '%s' field; it must be republished with 'dependencies' and 'resources' before it can be extended", ref.Name, ref.Version, key)
		}
	}

	merged := maps.Clone(parent)
	delete(merged, "source_url")
	for key, value := range child {
		if !slices.Contains(childMapKeys, key) {
			merged[key] = value
		}
	}
	merged[ExtendsKey] = ref.toMap()

	deps, err := mergeDependencies(parent, child)
	if err != nil {
		return Merge{}, err
	}
	if len(deps) > 0 {
		merged["dependencies"] = deps
	}

	parentResources, _ := parent["resources"].(map[string]any)
	childResources, _ := child["resources"].(map[string]any)
	resources, err := mergeDisjoint(parentResources, childResources, "resource")
	if err != nil {
		return Merge{}, err
	}
	if len(resources) > 0 {
		merged["resources"] = resources
	}

	steps, stepResources, warnings, err := mergeSteps(parent, child, childResources)
	if err != nil {
		return Merge{}, err
	}
	if steps != nil {
		merged["steps"] = steps
	}

	app, appWarnings, err := mergeApp(parent, child, ref)
	if err != nil {
		return Merge{}, err
	}
	if app != nil {
		merged["app"] = app
	}

	envWarnings, err := checkEnvReferences(merged)
	if err != nil {
		return Merge{}, err
	}

	warnings = append(warnings, appWarnings...)
	return Merge{Spec: merged, StepResources: stepResources, Warnings: append(warnings, envWarnings...)}, nil
}

func validateChildKeys(child map[string]any) error {
	var invalid []string
	for key, value := range child {
		if slices.Contains(childMapKeys, key) {
			continue
		}
		switch value.(type) {
		case map[string]any, []any:
			invalid = append(invalid, key)
		}
	}
	if len(invalid) == 0 {
		return nil
	}
	sort.Strings(invalid)
	return fmt.Errorf("a bundle that extends another may only set name, description, version, source_url, app, dependencies, resources, and steps; remove: %s", strings.Join(invalid, ", "))
}

func mergeSteps(parent, child, childResources map[string]any) ([]any, []StepResource, []string, error) {
	parentSteps, _ := parent["steps"].([]any)
	childSteps, _ := child["steps"].([]any)

	stepIndex := indexSteps(parentSteps)
	merged := slices.Clone(parentSteps)
	var bindings []StepResource
	var warnings []string
	bound := map[string]string{}

	for _, s := range childSteps {
		path, templates, err := validateChildStep(s)
		if err != nil {
			return nil, nil, nil, err
		}
		i, exists := stepIndex[path]
		if !exists {
			return nil, nil, nil, fmt.Errorf("step '%s' is not a step of the parent bundle", path)
		}
		if !filepath.IsLocal(path) {
			return nil, nil, nil, fmt.Errorf("step path '%s' must be relative to the bundle", path)
		}

		for _, field := range sortedKeys(templates) {
			binding, warning, bindErr := bindStepResource(path, field, templates[field], childResources, bound)
			if bindErr != nil {
				return nil, nil, nil, bindErr
			}
			if warning != "" {
				warnings = append(warnings, warning)
			}
			bound[field] = path
			bindings = append(bindings, binding)
		}

		parentStep, _ := merged[i].(map[string]any)
		if merged[i], err = addStepTemplates(parentStep, path, templates); err != nil {
			return nil, nil, nil, err
		}
	}

	for _, field := range sortedKeys(childResources) {
		if _, ok := bound[field]; !ok {
			return nil, nil, nil, fmt.Errorf("resource '%s' is not emitted by any step; add a jq template for it under steps[].resources", field)
		}
	}

	return merged, bindings, warnings, nil
}

func addStepTemplates(step map[string]any, path string, templates map[string]any) (map[string]any, error) {
	existing, _ := step["resources"].(map[string]any)
	combined, err := mergeDisjoint(existing, templates, fmt.Sprintf("steps[%s].resources", path))
	if err != nil {
		return nil, err
	}
	if len(combined) == 0 {
		return step, nil
	}
	merged := maps.Clone(step)
	merged["resources"] = combined
	return merged, nil
}

func indexSteps(steps []any) map[string]int {
	index := map[string]int{}
	for i, s := range steps {
		step, _ := s.(map[string]any)
		if path, ok := step["path"].(string); ok {
			index[path] = i
		}
	}
	return index
}

func validateChildStep(s any) (string, map[string]any, error) {
	step, ok := s.(map[string]any)
	if !ok {
		return "", nil, errors.New("each entry in 'steps' must be a map with a 'path'")
	}
	var invalid []string
	for _, key := range sortedKeys(step) {
		if !slices.Contains(childStepKeys, key) {
			invalid = append(invalid, key)
		}
	}
	if len(invalid) > 0 {
		return "", nil, fmt.Errorf("a step in a bundle that extends another may only set path and resources; remove: %s", strings.Join(invalid, ", "))
	}
	path, _ := step["path"].(string)
	templates, _ := step["resources"].(map[string]any)
	return path, templates, nil
}

func bindStepResource(path, field string, template any, childResources map[string]any, bound map[string]string) (StepResource, string, error) {
	if !resourceFieldRegex.MatchString(field) {
		return StepResource{}, "", fmt.Errorf("steps[%s].resources '%s' must contain only letters, digits, '_', and '-'", path, field)
	}
	if _, declared := childResources[field]; !declared {
		return StepResource{}, "", fmt.Errorf("steps[%s].resources '%s' is not declared in the bundle's 'resources'", path, field)
	}
	if other, dup := bound[field]; dup {
		return StepResource{}, "", fmt.Errorf("resource '%s' is emitted by both step '%s' and step '%s'", field, other, path)
	}
	query, _ := template.(string)
	if strings.TrimSpace(query) == "" {
		return StepResource{}, "", fmt.Errorf("steps[%s].resources '%s' must be a jq template", path, field)
	}
	var warning string
	if strings.Contains(query, ".dependencies.") {
		warning = fmt.Sprintf("steps[%s].resources.%s reads .dependencies; resource templates receive dependencies as .connections", path, field)
	}
	return StepResource{Step: path, Field: field, Query: query}, warning, nil
}

func mergeDependencies(parent, child map[string]any) (map[string]any, error) {
	parentDeps, _ := parent["dependencies"].(map[string]any)
	childDeps, _ := child["dependencies"].(map[string]any)
	paramNames := paramProperties(parent)

	merged := maps.Clone(parentDeps)
	if merged == nil {
		merged = map[string]any{}
	}
	for _, name := range sortedKeys(childDeps) {
		if _, exists := parentDeps[name]; exists {
			return nil, fmt.Errorf("dependency '%s' is already declared by the parent bundle; choose another name", name)
		}
		if _, exists := paramNames[name]; exists {
			return nil, fmt.Errorf("dependency '%s' collides with a parent param of the same name; choose another name", name)
		}
		merged[name] = childDeps[name]
	}
	return merged, nil
}

func mergeApp(parent, child map[string]any, ref ParentRef) (map[string]any, []string, error) {
	parentApp, parentHasApp := parent["app"].(map[string]any)
	childApp, childHasApp := child["app"].(map[string]any)

	var warnings []string
	if !parentHasApp {
		if childHasApp {
			warnings = append(warnings, fmt.Sprintf("parent %s@%s has no 'app' block, so its IaC likely ignores app envs and secrets; the child's 'app' block may have no effect", ref.Name, ref.Version))
		}
		return childApp, warnings, nil
	}
	if !childHasApp {
		return parentApp, warnings, nil
	}

	merged := maps.Clone(parentApp)
	for _, section := range []string{"envs", "secrets"} {
		combined, err := mergeDisjoint(parentApp[section], childApp[section], "app."+section)
		if err != nil {
			return nil, nil, err
		}
		if len(combined) > 0 {
			merged[section] = combined
		}
	}

	policies, _ := parentApp["policies"].([]any)
	childPolicies, _ := childApp["policies"].([]any)
	for _, p := range childPolicies {
		if !slices.Contains(policies, p) {
			policies = append(policies, p)
		}
	}
	if policies != nil {
		merged["policies"] = policies
	}

	return merged, warnings, nil
}

func mergeDisjoint(parentVal, childVal any, label string) (map[string]any, error) {
	parentMap, _ := parentVal.(map[string]any)
	childMap, _ := childVal.(map[string]any)
	merged := maps.Clone(parentMap)
	if merged == nil {
		merged = map[string]any{}
	}
	for _, key := range sortedKeys(childMap) {
		if _, exists := parentMap[key]; exists {
			return nil, fmt.Errorf("%s '%s' is already defined by the parent bundle; choose another name", label, key)
		}
		merged[key] = childMap[key]
	}
	return merged, nil
}

// A missing field makes the deployment fail, and so does an optional
// dependency that isn't connected, so both are caught here.
func checkEnvReferences(merged map[string]any) ([]string, error) {
	app, _ := merged["app"].(map[string]any)
	envs, _ := app["envs"].(map[string]any)
	params := paramProperties(merged)
	deps, _ := merged["dependencies"].(map[string]any)

	var warnings []string
	for _, envName := range sortedKeys(envs) {
		query, _ := envs[envName].(string)
		for _, match := range envReferenceRegex.FindAllStringSubmatch(query, -1) {
			source, field := match[1], match[2]
			if source == "params" {
				if field == "md_metadata" {
					continue
				}
				if _, ok := params[field]; !ok {
					return nil, fmt.Errorf("app.envs.%s references .params.%s, which is not a param of the merged bundle", envName, field)
				}
				continue
			}
			dep, ok := deps[field].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("app.envs.%s references .%s.%s, which is not a dependency of the merged bundle", envName, source, field)
			}
			if required, _ := dep["required"].(bool); !required {
				warnings = append(warnings, fmt.Sprintf("app.envs.%s reads optional dependency '%s'; deployments fail while it is not connected", envName, field))
			}
		}
	}
	return warnings, nil
}

func paramProperties(spec map[string]any) map[string]any {
	params, _ := spec["params"].(map[string]any)
	props, _ := params["properties"].(map[string]any)
	if props == nil {
		return map[string]any{}
	}
	return props
}

func sortedKeys(m map[string]any) []string {
	keys := slices.Collect(maps.Keys(m))
	sort.Strings(keys)
	return keys
}

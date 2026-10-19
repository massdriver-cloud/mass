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

// RuntimeKey is the key under `app` naming the runtime an application runs on.
const RuntimeKey = "runtime"

// ProvidesKey lists what a bundle provides, from ProvidesValues.
const ProvidesKey = "provides"

// ProvidesRuntime is the `provides` value that marks a bundle as a runtime.
const ProvidesRuntime = "runtime"

// ProvidesValues is the closed set a bundle may list in `provides`.
var ProvidesValues = []string{"runtime"}

// DocFiles are the top-level files an application carries over its runtime's copy.
var DocFiles = []string{"README.md", "readme.md", "CHANGELOG.md", "icon.svg", "operator.md", "operator.mdx"}

var appMergeKeys = []string{"app", "dependencies", "provides", "resources", "steps"}

var appScalarKeys = []string{"name", "description", "version", "source_url", "schema"}

var appStepKeys = []string{"path", "resources"}

var stepResourceKeys = []string{"template"}

var appSectionKeys = []string{"runtime", "envs", "secrets"}

var legacyInputKeys = []string{"connections", "artifacts"}

var envReferenceRegex = regexp.MustCompile(`\.(params|dependencies|connections)\.([A-Za-z0-9_-]+)`)

// RuntimeRef identifies the exact runtime an application was merged with. Its
// pinned form replaces the application's `app.runtime` constraint in the merged
// massdriver.yaml.
type RuntimeRef struct {
	Name       string
	Constraint string
	Version    string
	Digest     string
}

func (r RuntimeRef) pinned() string {
	return r.Name + "@" + r.Version
}

// ReadSpec reads massdriver.yaml from dir as an untyped document.
func ReadSpec(dir string) (map[string]any, error) {
	spec := map[string]any{}
	if err := files.Read(filepath.Join(dir, "massdriver.yaml"), &spec); err != nil {
		return nil, err
	}
	return spec, nil
}

// AppRuntime returns the `app.runtime` value of a massdriver.yaml document.
func AppRuntime(spec map[string]any) (any, bool) {
	app, _ := spec["app"].(map[string]any)
	raw, ok := app[RuntimeKey]
	return raw, ok
}

// ProvidesRuntimeValue reports whether a massdriver.yaml document lists
// `runtime` in `provides`.
func ProvidesRuntimeValue(spec map[string]any) bool {
	provides, _ := spec[ProvidesKey].([]any)
	return slices.Contains(provides, any(ProvidesRuntime))
}

// ParseAppRuntime splits an application's `app.runtime` value into a runtime
// name and version constraint. A bare name tracks `latest`.
func ParseAppRuntime(spec map[string]any) (name, constraint string, found bool, err error) {
	raw, ok := AppRuntime(spec)
	if !ok {
		return "", "", false, nil
	}
	ref, isString := raw.(string)
	if !isString || ref == "" {
		return "", "", true, errors.New("'app.runtime' must be a string like <runtime>@<constraint>, e.g. lambda@~1.2")
	}
	name, constraint, hasConstraint := strings.Cut(ref, "@")
	if !hasConstraint {
		constraint = "latest"
	}
	if strings.Contains(name, "/") {
		return "", "", true, fmt.Errorf("'app.runtime: %s' references another organization; a runtime must be in your organization", ref)
	}
	if name == "" || constraint == "" {
		return "", "", true, fmt.Errorf("'app.runtime: %s' must be <runtime>@<constraint>, e.g. lambda@~1.2", ref)
	}
	return name, constraint, true, nil
}

// CheckRuntime reports whether a pulled runtime can be run on: it must provide
// `runtime`, not itself run on a runtime, and use the current input format.
func CheckRuntime(runtime map[string]any, ref RuntimeRef) error {
	if _, ok := AppRuntime(runtime); ok {
		return fmt.Errorf("%s@%s runs on a runtime itself; applications cannot run on other applications", ref.Name, ref.Version)
	}
	if !ProvidesRuntimeValue(runtime) {
		return fmt.Errorf("%s@%s does not provide a runtime. Add `runtime` to `provides` in its massdriver.yaml and republish", ref.Name, ref.Version)
	}
	for _, key := range legacyInputKeys {
		if _, ok := runtime[key]; ok {
			return fmt.Errorf("runtime %s@%s uses the deprecated '%s' field; it must be republished with 'dependencies' and 'resources' before applications can run on it", ref.Name, ref.Version, key)
		}
	}
	return nil
}

// MergeApplication overlays an application's massdriver.yaml onto its runtime's.
// The runtime owns everything that shapes the IaC; the application supplies
// identity, extra dependencies, resources emitted by jq templates on the
// runtime's steps, and app wiring. The returned warnings are non-fatal.
func MergeApplication(runtime, application map[string]any, ref RuntimeRef) (map[string]any, []string, error) {
	version, ok := application["version"]
	if !ok {
		return nil, nil, errors.New("a bundle with 'app.runtime' must set its own 'version'")
	}
	if v, _ := version.(string); !validSemverRegex.MatchString(v) {
		return nil, nil, fmt.Errorf("invalid version in massdriver.yaml: %v. Version must follow semantic versioning (MAJOR.MINOR.PATCH), e.g., 1.2.3", version)
	}
	if err := validateApplicationKeys(application); err != nil {
		return nil, nil, err
	}
	if ProvidesRuntimeValue(application) {
		return nil, nil, errors.New("a bundle cannot both provide `runtime` and set `app.runtime`; applications cannot run on other applications")
	}

	merged := maps.Clone(runtime)
	delete(merged, "source_url")
	delete(merged, ProvidesKey)
	for key, value := range application {
		if !slices.Contains(appMergeKeys, key) {
			merged[key] = value
		}
	}
	if provides, ok := application[ProvidesKey]; ok {
		merged[ProvidesKey] = provides
	}

	deps, err := mergeDependencies(runtime, application)
	if err != nil {
		return nil, nil, err
	}
	if len(deps) > 0 {
		merged["dependencies"] = deps
	}

	runtimeResources, _ := runtime["resources"].(map[string]any)
	appResources, _ := application["resources"].(map[string]any)
	resources, err := mergeDisjoint(runtimeResources, appResources, "resource")
	if err != nil {
		return nil, nil, err
	}
	if len(resources) > 0 {
		merged["resources"] = resources
	}

	steps, err := mergeSteps(runtime, application, appResources, ref)
	if err != nil {
		return nil, nil, err
	}
	if steps != nil {
		merged["steps"] = steps
	}

	app, err := mergeApp(runtime, application, ref)
	if err != nil {
		return nil, nil, err
	}
	merged["app"] = app

	warnings, err := checkEnvReferences(merged)
	if err != nil {
		return nil, nil, err
	}

	return merged, warnings, nil
}

func validateApplicationKeys(application map[string]any) error {
	var invalid []string
	for _, key := range sortedKeys(application) {
		switch {
		case slices.Contains(appMergeKeys, key):
		case slices.Contains(appScalarKeys, key):
			if _, ok := application[key].(string); !ok {
				return fmt.Errorf("'%s' must be a string", key)
			}
		default:
			invalid = append(invalid, key)
		}
	}
	if len(invalid) == 0 {
		return nil
	}
	return fmt.Errorf("a bundle with 'app.runtime' may only set name, description, version, source_url, schema, provides, app, dependencies, resources, and steps; remove: %s", strings.Join(invalid, ", "))
}

func mergeSteps(runtime, application, appResources map[string]any, ref RuntimeRef) ([]any, error) {
	runtimeSteps, _ := runtime["steps"].([]any)
	appSteps, _ := application["steps"].([]any)

	stepIndex := indexSteps(runtimeSteps)
	merged := slices.Clone(runtimeSteps)
	bound := map[string]string{}

	for _, s := range appSteps {
		path, templates, err := validateApplicationStep(s)
		if err != nil {
			return nil, err
		}
		i, exists := stepIndex[path]
		if !exists {
			return nil, fmt.Errorf("step '%s' is not a step of runtime %s", path, ref.pinned())
		}
		for _, field := range sortedKeys(templates) {
			if err = checkStepTemplate(path, field, templates[field], appResources, bound); err != nil {
				return nil, err
			}
			bound[field] = path
		}

		runtimeStep, _ := merged[i].(map[string]any)
		if merged[i], err = addStepTemplates(runtimeStep, path, templates); err != nil {
			return nil, err
		}
	}

	for _, field := range sortedKeys(appResources) {
		if _, ok := bound[field]; !ok {
			return nil, fmt.Errorf("resource '%s' is not emitted by any step; add a jq template for it under steps[].resources.%s.template", field, field)
		}
	}

	return merged, nil
}

func checkStepTemplate(path, field string, entry any, appResources map[string]any, bound map[string]string) error {
	if _, declared := appResources[field]; !declared {
		return fmt.Errorf("steps[%s].resources '%s' is not declared in the bundle's 'resources'", path, field)
	}
	if other, dup := bound[field]; dup {
		return fmt.Errorf("resource '%s' is emitted by both step '%s' and step '%s'", field, other, path)
	}
	settings, ok := entry.(map[string]any)
	if !ok {
		return fmt.Errorf("steps[%s].resources '%s' must be a map with a 'template'", path, field)
	}
	var invalid []string
	for _, key := range sortedKeys(settings) {
		if !slices.Contains(stepResourceKeys, key) {
			invalid = append(invalid, key)
		}
	}
	if len(invalid) > 0 {
		return fmt.Errorf("steps[%s].resources '%s' may only set template; remove: %s", path, field, strings.Join(invalid, ", "))
	}
	if query, _ := settings["template"].(string); strings.TrimSpace(query) == "" {
		return fmt.Errorf("steps[%s].resources '%s' template must be a jq template", path, field)
	}
	return nil
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

func validateApplicationStep(s any) (string, map[string]any, error) {
	step, ok := s.(map[string]any)
	if !ok {
		return "", nil, errors.New("each entry in 'steps' must be a map with a 'path'")
	}
	var invalid []string
	for _, key := range sortedKeys(step) {
		if !slices.Contains(appStepKeys, key) {
			invalid = append(invalid, key)
		}
	}
	if len(invalid) > 0 {
		return "", nil, fmt.Errorf("a step in a bundle with 'app.runtime' may only set path and resources; remove: %s", strings.Join(invalid, ", "))
	}
	path, _ := step["path"].(string)
	templates, _ := step["resources"].(map[string]any)
	return path, templates, nil
}

func mergeDependencies(runtime, application map[string]any) (map[string]any, error) {
	runtimeDeps, _ := runtime["dependencies"].(map[string]any)
	appDeps, _ := application["dependencies"].(map[string]any)
	paramNames := paramProperties(runtime)

	merged := maps.Clone(runtimeDeps)
	if merged == nil {
		merged = map[string]any{}
	}
	for _, name := range sortedKeys(appDeps) {
		if _, exists := runtimeDeps[name]; exists {
			return nil, fmt.Errorf("dependency '%s' is already declared by the runtime; choose another name", name)
		}
		if _, exists := paramNames[name]; exists {
			return nil, fmt.Errorf("dependency '%s' collides with a runtime param of the same name; choose another name", name)
		}
		merged[name] = appDeps[name]
	}
	return merged, nil
}

func mergeApp(runtime, application map[string]any, ref RuntimeRef) (map[string]any, error) {
	runtimeApp, _ := runtime["app"].(map[string]any)
	applicationApp, _ := application["app"].(map[string]any)

	var invalid []string
	for _, key := range sortedKeys(applicationApp) {
		if !slices.Contains(appSectionKeys, key) {
			invalid = append(invalid, key)
		}
	}
	if len(invalid) > 0 {
		return nil, fmt.Errorf("a bundle with 'app.runtime' may only set runtime, envs, and secrets under 'app'; remove: %s", strings.Join(invalid, ", "))
	}

	merged := maps.Clone(runtimeApp)
	if merged == nil {
		merged = map[string]any{}
	}
	merged[RuntimeKey] = ref.pinned()
	for _, section := range []string{"envs", "secrets"} {
		combined, err := mergeDisjoint(runtimeApp[section], applicationApp[section], "app."+section)
		if err != nil {
			return nil, err
		}
		if len(combined) > 0 {
			merged[section] = combined
		}
	}

	return merged, nil
}

func mergeDisjoint(runtimeVal, appVal any, label string) (map[string]any, error) {
	runtimeMap, _ := runtimeVal.(map[string]any)
	appMap, _ := appVal.(map[string]any)
	merged := maps.Clone(runtimeMap)
	if merged == nil {
		merged = map[string]any{}
	}
	for _, key := range sortedKeys(appMap) {
		if _, exists := runtimeMap[key]; exists {
			return nil, fmt.Errorf("%s '%s' is already defined by the runtime; choose another name", label, key)
		}
		merged[key] = appMap[key]
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

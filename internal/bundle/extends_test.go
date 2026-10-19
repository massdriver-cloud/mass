package bundle_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"sigs.k8s.io/yaml"
)

func mustYAML(t *testing.T, doc string) map[string]any {
	t.Helper()
	out := map[string]any{}
	if err := yaml.Unmarshal([]byte(doc), &out); err != nil {
		t.Fatalf("parsing yaml: %v", err)
	}
	return out
}

func TestParseExtends(t *testing.T) {
	tests := []struct {
		name           string
		spec           string
		wantName       string
		wantConstraint string
		wantFound      bool
		wantErr        string
	}{
		{name: "absent", spec: "name: x", wantFound: false},
		{name: "with constraint", spec: "app: {runtime: lambda@~1.2}", wantName: "lambda", wantConstraint: "~1.2", wantFound: true},
		{name: "dev constraint", spec: "app: {runtime: lambda@~1.2+dev}", wantName: "lambda", wantConstraint: "~1.2+dev", wantFound: true},
		{name: "bare name tracks latest", spec: "app: {runtime: lambda}", wantName: "lambda", wantConstraint: "latest", wantFound: true},
		{name: "other organization", spec: "app: {runtime: acme/lambda@~1}", wantFound: true, wantErr: "another organization"},
		{name: "not a string", spec: "app: {runtime: {name: lambda}}", wantFound: true, wantErr: "must be a string"},
		{name: "empty constraint", spec: "app: {runtime: lambda@}", wantFound: true, wantErr: "must be <runtime>@<constraint>"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, constraint, found, err := bundle.ParseExtends(mustYAML(t, tc.spec))
			if found != tc.wantFound {
				t.Fatalf("found = %v, want %v", found, tc.wantFound)
			}
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if name != tc.wantName || constraint != tc.wantConstraint {
				t.Errorf("got %s@%s, want %s@%s", name, constraint, tc.wantName, tc.wantConstraint)
			}
		})
	}
}

const parentSpec = `
name: lambda
description: A lambda function.
version: 1.2.7
source_url: https://github.com/acme/lambda
provides: [runtime]
steps:
  - path: src
    provisioner: opentofu
params:
  required: [memory]
  properties:
    memory:
      type: integer
ui:
  ui:order: ["*"]
dependencies:
  network:
    resource_type: aws-vpc@~1
    required: true
resources:
  function:
    resource_type: aws-lambda@1.0.0
    required: true
app:
  envs:
    MEMORY: .params.memory | tostring
  secrets:
    SHARED:
      required: false
`

func TestCheckRuntime(t *testing.T) {
	ref := bundle.ParentRef{Name: "lambda", Version: "1.2.7"}

	tests := []struct {
		name    string
		parent  string
		wantErr string
	}{
		{name: "runtime", parent: parentSpec},
		{
			name:    "bundle that is not a runtime",
			parent:  "name: lambda\napp:\n  envs: {}\n",
			wantErr: "lambda@1.2.7 does not provide a runtime. Add `runtime` to `provides`",
		},
		{
			name:    "bundle with an empty provides",
			parent:  "name: lambda\nprovides: []\napp:\n  envs: {}\n",
			wantErr: "does not provide a runtime",
		},
		{
			name:   "runtime without an app block",
			parent: "name: lambda\nprovides: [runtime]\n",
		},
		{
			name:    "application",
			parent:  "name: lambda\nprovides: [runtime]\napp:\n  envs: {}\n  runtime: base@0.0.1\n",
			wantErr: "lambda@1.2.7 runs on a runtime itself",
		},
		{
			name:    "runtime uses legacy connections",
			parent:  "name: lambda\nprovides: [runtime]\napp:\n  envs: {}\nconnections:\n  properties: {}\n",
			wantErr: "deprecated 'connections'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := bundle.CheckRuntime(mustYAML(t, tc.parent), ref)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestMergeExtends(t *testing.T) {
	ref := bundle.ParentRef{Name: "lambda", Constraint: "~1.2", Version: "1.2.7", Digest: "sha256:abc"}

	tests := []struct {
		name         string
		child        string
		parent       string
		wantErr      string
		wantWarnings []string
		check        func(t *testing.T, merged map[string]any)
	}{
		{
			name: "child identity and additions overlay the parent",
			child: `
name: payments-api
description: Handles card captures.
version: 0.1.0
provides: []
dependencies:
  db:
    resource_type: postgres@~1
    required: true
app:
  runtime: lambda@~1.2
  envs:
    DB_HOST: .dependencies.db.host
  secrets:
    STRIPE_KEY:
      required: true
`,
			check: func(t *testing.T, merged map[string]any) {
				want := mustYAML(t, `
name: payments-api
description: Handles card captures.
version: 0.1.0
provides: []
steps:
  - path: src
    provisioner: opentofu
params:
  required: [memory]
  properties:
    memory:
      type: integer
ui:
  ui:order: ["*"]
dependencies:
  network:
    resource_type: aws-vpc@~1
    required: true
  db:
    resource_type: postgres@~1
    required: true
resources:
  function:
    resource_type: aws-lambda@1.0.0
    required: true
app:
  runtime: lambda@1.2.7
  envs:
    MEMORY: .params.memory | tostring
    DB_HOST: .dependencies.db.host
  secrets:
    SHARED:
      required: false
    STRIPE_KEY:
      required: true
`)
				if !reflect.DeepEqual(merged, want) {
					got, _ := yaml.Marshal(merged)
					t.Errorf("merged document mismatch:\n%s", got)
				}
			},
		},
		{
			name:  "child source_url replaces the parent's, absent drops it",
			child: "name: app\nversion: 0.1.0\n",
			check: func(t *testing.T, merged map[string]any) {
				if _, ok := merged["source_url"]; ok {
					t.Errorf("source_url = %v, want absent", merged["source_url"])
				}
				if merged["description"] != "A lambda function." {
					t.Errorf("description = %v, want the parent's", merged["description"])
				}
			},
		},
		{
			name:   "runtime without an app block",
			child:  "name: app\nversion: 0.1.0\napp:\n  runtime: lambda@~1.2\n  envs:\n    STAGE: '@text \"prod\"'\n",
			parent: "name: lambda\nprovides: [runtime]\n",
			check: func(t *testing.T, merged map[string]any) {
				want := mustYAML(t, "runtime: lambda@1.2.7\nenvs:\n  STAGE: '@text \"prod\"'\n")
				if !reflect.DeepEqual(merged["app"], want) {
					t.Errorf("app = %v, want %v", merged["app"], want)
				}
			},
		},
		{
			name:    "child app sets a key other than envs and secrets",
			child:   "name: app\nversion: 0.1.0\napp:\n  policies:\n    - .dependencies.network.data.read\n",
			wantErr: "may only set runtime, envs, and secrets under 'app'; remove: policies",
		},
		{
			name:    "child sets a parent-owned key",
			child:   "name: app\nversion: 0.1.0\nparams:\n  properties: {}\nui: {}\n",
			wantErr: "remove: params, ui",
		},
		{
			name: "child resource is emitted by a parent step",
			child: `
name: billing-api
version: 0.1.0
resources:
  billing:
    resource_type: billing-api@~1
    required: true
steps:
  - path: src
    resources:
      billing:
        template: '{url: .outputs.url}'
`,
			check: func(t *testing.T, merged map[string]any) {
				want := mustYAML(t, `
steps:
  - path: src
    provisioner: opentofu
    resources:
      billing:
        template: '{url: .outputs.url}'
resources:
  function:
    resource_type: aws-lambda@1.0.0
    required: true
  billing:
    resource_type: billing-api@~1
    required: true
`)
				if !reflect.DeepEqual(merged["steps"], want["steps"]) || !reflect.DeepEqual(merged["resources"], want["resources"]) {
					t.Errorf("steps = %v, resources = %v", merged["steps"], merged["resources"])
				}
			},
		},
		{
			name:    "child step overrides the provisioner",
			child:   "name: app\nversion: 0.1.0\nsteps:\n  - path: src\n    provisioner: helm\n",
			wantErr: "may only set path and resources; remove: provisioner",
		},
		{
			name:    "child step names a step the parent lacks",
			child:   "name: app\nversion: 0.1.0\nresources:\n  billing:\n    resource_type: billing-api@~1\nsteps:\n  - path: chart\n    resources:\n      billing:\n        template: .outputs\n",
			wantErr: "step 'chart' is not a step of the parent bundle",
		},
		{
			name:    "resource collides with a parent resource",
			child:   "name: app\nversion: 0.1.0\nresources:\n  function:\n    resource_type: aws-lambda@1.0.0\n",
			wantErr: "resource 'function' is already defined by the parent bundle",
		},
		{
			name:    "resource no step emits",
			child:   "name: app\nversion: 0.1.0\nresources:\n  billing:\n    resource_type: billing-api@~1\n",
			wantErr: "resource 'billing' is not emitted by any step",
		},
		{
			name:    "step template for an undeclared resource",
			child:   "name: app\nversion: 0.1.0\nsteps:\n  - path: src\n    resources:\n      billing:\n        template: .outputs\n",
			wantErr: "steps[src].resources 'billing' is not declared",
		},
		{
			name:    "step template is empty",
			child:   "name: app\nversion: 0.1.0\nresources:\n  billing:\n    resource_type: billing-api@~1\nsteps:\n  - path: src\n    resources:\n      billing:\n        template: ''\n",
			wantErr: "steps[src].resources 'billing' template must be a jq template",
		},
		{
			name:    "step resource is a bare template",
			child:   "name: app\nversion: 0.1.0\nresources:\n  billing:\n    resource_type: billing-api@~1\nsteps:\n  - path: src\n    resources:\n      billing: .outputs\n",
			wantErr: "steps[src].resources 'billing' must be a map with a 'template'",
		},
		{
			name:    "step resource sets an unknown key",
			child:   "name: app\nversion: 0.1.0\nresources:\n  billing:\n    resource_type: billing-api@~1\nsteps:\n  - path: src\n    resources:\n      billing:\n        template: .outputs\n        foo: bar\n",
			wantErr: "steps[src].resources 'billing' may only set template; remove: foo",
		},
		{
			name:    "child omits version",
			child:   "name: app\n",
			wantErr: "must set its own 'version'",
		},
		{
			name:  "runtime's provides is not inherited",
			child: "name: app\nversion: 0.1.0\n",
			check: func(t *testing.T, merged map[string]any) {
				if _, ok := merged["provides"]; ok {
					t.Errorf("provides = %v, want absent", merged["provides"])
				}
			},
		},
		{
			name:    "child provides runtime",
			child:   "name: app\nversion: 0.1.0\nprovides: [runtime]\n",
			wantErr: "cannot both provide `runtime` and set `app.runtime`",
		},
		{
			name:    "dependency collides with parent dependency",
			child:   "name: app\nversion: 0.1.0\ndependencies:\n  network:\n    resource_type: aws-vpc@~1\n    required: true\n",
			wantErr: "dependency 'network' is already declared",
		},
		{
			name:    "dependency collides with parent param",
			child:   "name: app\nversion: 0.1.0\ndependencies:\n  memory:\n    resource_type: aws-vpc@~1\n    required: true\n",
			wantErr: "collides with a parent param",
		},
		{
			name:    "env collides with parent env",
			child:   "name: app\nversion: 0.1.0\napp:\n  envs:\n    MEMORY: .params.memory | tostring\n",
			wantErr: "app.envs 'MEMORY' is already defined",
		},
		{
			name:    "secret collides with parent secret",
			child:   "name: app\nversion: 0.1.0\napp:\n  secrets:\n    SHARED:\n      required: true\n",
			wantErr: "app.secrets 'SHARED' is already defined",
		},
		{
			name:    "env references a missing param",
			child:   "name: app\nversion: 0.1.0\napp:\n  envs:\n    REGION: .params.region\n",
			wantErr: "references .params.region",
		},
		{
			name:    "env references a missing dependency",
			child:   "name: app\nversion: 0.1.0\napp:\n  envs:\n    DB: .dependencies.db.data.host\n",
			wantErr: "references .dependencies.db",
		},
		{
			name:         "env reads an optional dependency",
			child:        "name: app\nversion: 0.1.0\ndependencies:\n  cache:\n    resource_type: redis@~1\n    required: false\napp:\n  envs:\n    CACHE: .dependencies.cache.data.host\n",
			wantWarnings: []string{"app.envs.CACHE reads optional dependency 'cache'; deployments fail while it is not connected"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parent := parentSpec
			if tc.parent != "" {
				parent = tc.parent
			}
			merged, warnings, err := bundle.MergeExtends(mustYAML(t, parent), mustYAML(t, tc.child), ref)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(warnings, tc.wantWarnings) {
				t.Errorf("warnings = %q, want %q", warnings, tc.wantWarnings)
			}
			if tc.check != nil {
				tc.check(t, merged)
			}
		})
	}
}

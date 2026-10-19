package bundle_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/stretchr/testify/assert"
)

func TestLintSchema(t *testing.T) {
	type test struct {
		name string
		bun  *bundle.Bundle
		want bundle.LintResult
	}
	tests := []test{
		{
			name: "Valid pass",
			bun: &bundle.Bundle{
				Name:        "example",
				Description: "description",
				Type:        "infrastructure",
				Params:      map[string]any{"properties": map[string]any{}},
				Connections: map[string]any{"properties": map[string]any{}},
				Artifacts:   map[string]any{"properties": map[string]any{}},
				UI:          map[string]any{"properties": map[string]any{}},
			},
			want: bundle.LintResult{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bundleSchema, err := os.ReadFile("testdata/lint/schema/bundle.json")
			if err != nil {
				t.Fatalf("failed to read artifact definition schema: %v", err)
			}

			// Start mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/json-schemas/bundle.json":
					_, _ = w.Write(bundleSchema)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			got := tc.bun.LintSchema(server.URL)

			assert.Len(t, got.Issues, len(tc.want.Issues))
			for i := range tc.want.Issues {
				assert.Equal(t, tc.want.Issues[i].Rule, got.Issues[i].Rule)
				assert.Equal(t, tc.want.Issues[i].Severity, got.Issues[i].Severity)
				assert.Contains(t, got.Issues[i].Message, tc.want.Issues[i].Message)
			}
		})
	}
}

func TestLintParamsConnectionsNameCollision(t *testing.T) {
	type test struct {
		name string
		bun  *bundle.Bundle
		want bundle.LintResult
	}
	tests := []test{
		{
			name: "Valid Pass",
			bun: &bundle.Bundle{
				Params: map[string]any{
					"properties": map[string]any{
						"param": "foo",
					},
				},
				Connections: map[string]any{
					"properties": map[string]any{
						"connection": "foo",
					},
				},
			},
			want: bundle.LintResult{},
		},
		{
			name: "Invalid Error",
			bun: &bundle.Bundle{
				Params: map[string]any{
					"properties": map[string]any{
						"database": "foo",
					},
				},
				Connections: map[string]any{
					"properties": map[string]any{
						"database": "foo",
					},
				},
			},
			want: bundle.LintResult{
				Issues: []bundle.LintIssue{
					{
						Rule:     "name-collision",
						Severity: bundle.LintError,
						Message:  "a parameter and connection have the same name: database",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.bun.DereferenceSchemas(".", stubResolver(nil)); err != nil {
				t.Fatal(err)
			}
			got := tc.bun.LintParamsConnectionsNameCollision()

			assert.ElementsMatch(t, tc.want.Issues, got.Issues)
		})
	}
}

func TestLintInputsMatchProvisioner(t *testing.T) {
	{
		type test struct {
			name string
			bun  *bundle.Bundle
			want bundle.LintResult
		}
		tests := []test{
			{
				name: "Valid all params",
				bun: &bundle.Bundle{
					Name:        "example",
					Description: "description",
					Type:        "infrastructure",
					Steps: []bundle.Step{{
						Path:        "testdata/lint/module",
						Provisioner: "opentofu",
					}},
					Params: map[string]any{
						"properties": map[string]any{
							"foo": map[string]any{},
							"bar": map[string]any{},
						},
					},
					Connections: map[string]any{},
					Artifacts:   map[string]any{},
					UI:          map[string]any{},
				},
				want: bundle.LintResult{},
			}, {
				name: "Valid param and connection",
				bun: &bundle.Bundle{
					Name:        "example",
					Description: "description",
					Type:        "infrastructure",
					Steps: []bundle.Step{{
						Path:        "testdata/lint/module",
						Provisioner: "opentofu",
					}},
					Params: map[string]any{
						"properties": map[string]any{
							"foo": map[string]any{},
						},
					},
					Connections: map[string]any{
						"properties": map[string]any{
							"bar": map[string]any{},
						},
					},
					Artifacts: map[string]any{},
					UI:        map[string]any{},
				},
				want: bundle.LintResult{},
			}, {
				name: "Invalid missing massdriver input",
				bun: &bundle.Bundle{
					Name:        "example",
					Description: "description",
					Type:        "infrastructure",
					Steps: []bundle.Step{{
						Path:        "testdata/lint/module",
						Provisioner: "opentofu",
					}},
					Params: map[string]any{
						"properties": map[string]any{
							"foo": map[string]any{},
						},
					},
					Connections: map[string]any{},
					Artifacts:   map[string]any{},
					UI:          map[string]any{},
				},
				want: bundle.LintResult{
					Issues: []bundle.LintIssue{
						{
							Rule:     "param-mismatch",
							Severity: bundle.LintWarning,
							Message: `missing inputs detected in step testdata/lint/module:
	- input "bar" declared in IaC but missing massdriver.yaml declaration
`,
						},
					},
				},
			}, {
				name: "Invalid missing IaC input",
				bun: &bundle.Bundle{
					Name:        "example",
					Description: "description",
					Type:        "infrastructure",
					Steps: []bundle.Step{{
						Path:        "testdata/lint/module",
						Provisioner: "opentofu",
					}},
					Params: map[string]any{
						"properties": map[string]any{
							"foo": map[string]any{},
							"bar": map[string]any{},
							"baz": map[string]any{},
						},
					},
					Connections: map[string]any{},
					Artifacts:   map[string]any{},
					UI:          map[string]any{},
				},
				want: bundle.LintResult{
					Issues: []bundle.LintIssue{
						{
							Rule:     "param-mismatch",
							Severity: bundle.LintWarning,
							Message: `missing inputs detected in step testdata/lint/module:
	- input "baz" declared in massdriver.yaml but missing IaC declaration
`,
						},
					},
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.bun.DereferenceSchemas(".", stubResolver(nil)); err != nil {
					t.Fatal(err)
				}
				got := tc.bun.LintInputsMatchProvisioner()

				assert.ElementsMatch(t, tc.want.Issues, got.Issues)
			})
		}
	}
}

func TestLintMatchRequired(t *testing.T) {
	type test struct {
		name string
		bun  *bundle.Bundle
		want bundle.LintResult
	}
	tests := []test{
		{
			name: "Valid pass",
			bun: &bundle.Bundle{
				Name:        "example",
				Description: "description",
				Type:        "infrastructure",
				Params: map[string]any{
					"required": []any{"foo"},
					"properties": map[string]any{
						"foo": map[string]any{
							"type": "string",
						},
					},
				},
				Connections: map[string]any{},
				Artifacts:   map[string]any{},
				UI:          map[string]any{},
			},
			want: bundle.LintResult{},
		},
		{
			name: "Invalid missing param",
			bun: &bundle.Bundle{
				Name:        "example",
				Description: "description",
				Type:        "infrastructure",
				Params: map[string]any{
					"required": []any{"bar"},
					"properties": map[string]any{
						"foo": map[string]any{
							"type": "string",
						},
					},
				},
				Connections: map[string]any{},
				Artifacts:   map[string]any{},
				UI:          map[string]any{},
			},
			want: bundle.LintResult{
				Issues: []bundle.LintIssue{
					{
						Rule:     "required-match",
						Severity: bundle.LintError,
						Message:  "required parameter bar is not defined in properties",
					},
				},
			},
		},
		{
			name: "Nested valid test",
			bun: &bundle.Bundle{
				Name:        "example",
				Description: "description",
				Type:        "infrastructure",
				Params: map[string]any{
					"required": []any{"foo"},
					"properties": map[string]any{
						"foo": map[string]any{
							"type":     "object",
							"required": []any{"bar"},
							"properties": map[string]any{
								"bar": map[string]any{
									"type": "string",
								},
							},
						},
					},
				},
				Connections: map[string]any{},
				Artifacts:   map[string]any{},
				UI:          map[string]any{},
			},
			want: bundle.LintResult{},
		},
		{
			name: "Nested invalid test",
			bun: &bundle.Bundle{
				Name:        "example",
				Description: "description",
				Type:        "infrastructure",
				Params: map[string]any{
					"required": []any{"foo"},
					"properties": map[string]any{
						"foo": map[string]any{
							"type":     "object",
							"required": []any{"baz", "bar"},
							"properties": map[string]any{
								"bar": map[string]any{
									"type": "string",
								},
							},
						},
					},
				},
				Connections: map[string]any{},
				Artifacts:   map[string]any{},
				UI:          map[string]any{},
			},
			want: bundle.LintResult{
				Issues: []bundle.LintIssue{
					{
						Rule:     "required-match",
						Severity: bundle.LintError,
						Message:  "required parameter baz is not defined in properties",
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.bun.LintMatchRequired()

			assert.ElementsMatch(t, tc.want.Issues, got.Issues)
		})
	}
}

func TestLintProvides(t *testing.T) {
	resources := map[string]bundle.Resource{"function": {ResourceType: "aws-lambda@1.0.0"}}
	app := &bundle.AppSpec{}

	issue := func(message string) bundle.LintIssue {
		return bundle.LintIssue{Rule: "provides", Severity: bundle.LintError, Message: message}
	}

	tests := []struct {
		name string
		bun  *bundle.Bundle
		want []bundle.LintIssue
	}{
		{
			name: "provides is not set",
			bun:  &bundle.Bundle{},
		},
		{
			name: "empty provides",
			bun:  &bundle.Bundle{Provides: []string{}},
		},
		{
			name: "runtime with resources and an app block",
			bun:  &bundle.Bundle{Provides: []string{"runtime"}, Resources: resources, AppSpec: app},
		},
		{
			name: "value outside the set",
			bun:  &bundle.Bundle{Provides: []string{"compute", "runtime"}, Resources: resources, AppSpec: app},
			want: []bundle.LintIssue{issue("provides: 'compute' is not one of runtime")},
		},
		{
			name: "duplicate value",
			bun:  &bundle.Bundle{Provides: []string{"runtime", "runtime"}, Resources: resources, AppSpec: app},
			want: []bundle.LintIssue{issue("provides lists 'runtime' more than once")},
		},
		{
			name: "runtime without resources",
			bun:  &bundle.Bundle{Provides: []string{"runtime"}, AppSpec: app},
		},
		{
			name: "runtime without an app block",
			bun:  &bundle.Bundle{Provides: []string{"runtime"}, Resources: resources},
		},
		{
			name: "runtime that runs on a runtime",
			bun:  &bundle.Bundle{Provides: []string{"runtime"}, Resources: resources, AppSpec: &bundle.AppSpec{Runtime: "lambda@~1.2"}},
			want: []bundle.LintIssue{issue("A bundle cannot both provide `runtime` and set `app.runtime`; applications cannot run on other applications.")},
		},
		{
			name: "application without provides",
			bun:  &bundle.Bundle{Resources: resources, AppSpec: &bundle.AppSpec{Runtime: "lambda@~1.2"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.ElementsMatch(t, tc.want, tc.bun.LintProvides().Issues)
		})
	}
}

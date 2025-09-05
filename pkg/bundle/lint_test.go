package bundle_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/stretchr/testify/assert"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
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
				Schema:      "draft-07",
				Type:        "infrastructure",
				Params:      map[string]any{"properties": map[string]any{}},
				Connections: map[string]any{"properties": map[string]any{}},
				Artifacts:   map[string]any{"properties": map[string]any{}},
				UI:          map[string]any{"properties": map[string]any{}},
			},
			want: bundle.LintResult{},
		},
		{
			name: "Invalid missing schema field",
			bun: &bundle.Bundle{
				Name:        "example",
				Description: "description",
				Type:        "infrastructure",
				Params:      map[string]any{"properties": map[string]any{}},
				Connections: map[string]any{"properties": map[string]any{}},
				Artifacts:   map[string]any{"properties": map[string]any{}},
				UI:          map[string]any{"properties": map[string]any{}},
			},
			want: bundle.LintResult{
				Issues: []bundle.LintIssue{
					{
						Rule:     "schema-validation",
						Severity: bundle.LintError,
						Message:  "missing property 'schema'",
					},
				},
			},
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
					w.Write([]byte(bundleSchema))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			mdClient := client.Client{
				Config: config.Config{
					URL: server.URL,
				},
			}

			got := tc.bun.LintSchema(&mdClient)

			assert.Equal(t, len(tc.want.Issues), len(got.Issues))
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
					Schema:      "draft-07",
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
					Schema:      "draft-07",
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
				Schema:      "draft-07",
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
				Schema:      "draft-07",
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
				Schema:      "draft-07",
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
				Schema:      "draft-07",
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

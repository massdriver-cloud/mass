package bundle_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/xeipuuv/gojsonschema"
)

func TestLintSchema(t *testing.T) {
	type test struct {
		name string
		bun  *bundle.Bundle
		err  error
	}
	tests := []test{
		{
			name: "Valid pass",
			bun: &bundle.Bundle{
				Name:        "example",
				Description: "description",
				Schema:      "draft-07",
				Type:        "infrastructure",
				Params:      map[string]any{},
				Connections: map[string]any{},
				Artifacts:   map[string]any{},
				UI:          map[string]any{},
			},
			err: nil,
		},
		{
			name: "Invalid missing schema",
			bun: &bundle.Bundle{
				Name:        "example",
				Description: "description",
				Type:        "infrastructure",
				Params:      map[string]any{},
				Connections: map[string]any{},
				Artifacts:   map[string]any{},
				UI:          map[string]any{},
			},
			err: errors.New(`massdriver.yaml has schema violations:
	- schema: schema must be one of the following: "draft-07"
`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schemaLoader := gojsonschema.NewReferenceLoader("file://testdata/lint/schema/bundle.json")

			err := tc.bun.LintSchema(schemaLoader)
			if tc.err != nil {
				if err == nil {
					t.Errorf("expected an error, got nil")
				} else if tc.err.Error() != err.Error() {
					t.Errorf("got %v, want %v", err.Error(), tc.err.Error())
				}
			} else if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}
		})
	}
}

func TestLintParamsConnectionsNameCollision(t *testing.T) {
	type test struct {
		name string
		bun  *bundle.Bundle
		err  error
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
			err: nil,
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
			err: fmt.Errorf("a parameter and connection have the same name: database"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.bun.LintParamsConnectionsNameCollision()
			if tc.err != nil {
				if err == nil {
					t.Errorf("expected an error, got nil")
				} else if tc.err.Error() != err.Error() {
					t.Errorf("got %v, want %v", err.Error(), tc.err.Error())
				}
			} else if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}
		})
	}
}

func TestLintInputsMatchProvisioner(t *testing.T) {
	{
		type test struct {
			name string
			bun  *bundle.Bundle
			err  error
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
				err: nil,
			}, {
				name: "Valid param and connection",
				bun: &bundle.Bundle{
					Name:        "example",
					Description: "description",
					Schema:      "draft-07",
					Type:        "infrastructure",
					Steps: []bundle.Step{{
						Path:        "testdata/lintmodule",
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
				err: nil,
			}, {
				name: "Invalid missing massdriver input",
				bun: &bundle.Bundle{
					Name:        "example",
					Description: "description",
					Type:        "infrastructure",
					Steps: []bundle.Step{{
						Path:        "testdata/lintmodule",
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
				err: errors.New(`missing inputs detected in step testdata/lintmodule:
	- input "bar" declared in IaC but missing massdriver.yaml declaration
`),
			}, {
				name: "Invalid missing IaC input",
				bun: &bundle.Bundle{
					Name:        "example",
					Description: "description",
					Type:        "infrastructure",
					Steps: []bundle.Step{{
						Path:        "testdata/lintmodule",
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
				err: errors.New(`missing inputs detected in step testdata/lintmodule:
	- input "baz" declared in massdriver.yaml but missing IaC declaration
`),
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.bun.LintInputsMatchProvisioner()
				if tc.err != nil {
					if err == nil {
						t.Errorf("expected an error, got nil")
					} else if tc.err.Error() != err.Error() {
						t.Errorf("got %v, want %v", err.Error(), tc.err.Error())
					}
				} else if err != nil {
					t.Fatalf("%d, unexpected error", err)
				}
			})
		}
	}
}

func TestLintMatchRequired(t *testing.T) {
	type test struct {
		name string
		bun  *bundle.Bundle
		err  error
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
			err: nil,
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
			err: errors.New("required parameter bar is not defined in properties"),
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
			err: nil,
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
			err: errors.New("required parameter baz is not defined in properties"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.bun.LintMatchRequired()
			if tc.err != nil {
				if err == nil {
					t.Errorf("expected an error, got nil")
				} else if tc.err.Error() != err.Error() {
					t.Errorf("got %v, want %v", err.Error(), tc.err.Error())
				}
			} else if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}
		})
	}
}

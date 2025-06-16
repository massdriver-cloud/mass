package bundle_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
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
				Params:      map[string]interface{}{},
				Connections: map[string]interface{}{},
				Artifacts:   map[string]interface{}{},
				UI:          map[string]interface{}{},
			},
			err: nil,
		},
		{
			name: "Invalid missing schema",
			bun: &bundle.Bundle{
				Name:        "example",
				Description: "description",
				Type:        "infrastructure",
				Params:      map[string]interface{}{},
				Connections: map[string]interface{}{},
				Artifacts:   map[string]interface{}{},
				UI:          map[string]interface{}{},
			},
			err: errors.New(`massdriver.yaml has schema violations:
	- schema: schema must be one of the following: "draft-07"
`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.bun.LintSchema()
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
				Params: map[string]interface{}{
					"properties": map[string]interface{}{
						"param": "foo",
					},
				},
				Connections: map[string]interface{}{
					"properties": map[string]interface{}{
						"connection": "foo",
					},
				},
			},
			err: nil,
		},
		{
			name: "Invalid Error",
			bun: &bundle.Bundle{
				Params: map[string]interface{}{
					"properties": map[string]interface{}{
						"database": "foo",
					},
				},
				Connections: map[string]interface{}{
					"properties": map[string]interface{}{
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

func TestLintEnvs(t *testing.T) {
	type test struct {
		name string
		b    *bundle.Bundle
		want map[string]string
		err  error
	}
	tests := []test{
		{
			name: "params, connections, secrets working",
			b: &bundle.Bundle{
				AppSpec: &bundle.AppSpec{
					Envs: map[string]string{
						"FOO":        ".params.foo",
						"INTEGER":    ".params.int",
						"CONNECTION": ".connections.connection1",
						"SECRET":     ".secrets.shh",
					},
					Secrets: map[string]bundle.Secret{
						"shh": {},
					},
				},
				Params: map[string]interface{}{
					"properties": map[string]interface{}{
						"foo": map[string]interface{}{
							"type":  "string",
							"const": "bar",
						},
						"int": map[string]interface{}{
							"type":  "integer",
							"const": 4,
						},
					},
				},
				Connections: map[string]interface{}{
					"properties": map[string]interface{}{
						"connection1": map[string]interface{}{
							"type":  "string",
							"const": "whatever",
						},
					},
				},
			},
			want: map[string]string{
				"FOO":        "bar",
				"INTEGER":    "4",
				"CONNECTION": "whatever",
				"SECRET":     "some-secret-value",
			},
			err: nil,
		},
		{
			name: "error on missing data",
			b: &bundle.Bundle{
				AppSpec: &bundle.AppSpec{
					Envs: map[string]string{
						"FOO": ".params.foo",
					},
					Secrets: map[string]bundle.Secret{},
				},
				Params:      map[string]interface{}{},
				Connections: map[string]interface{}{},
			},
			want: map[string]string{},
			err:  errors.New("the jq query for environment variable FOO didn't produce a result"),
		},
		{
			name: "error on invalid jq syntax",
			b: &bundle.Bundle{
				AppSpec: &bundle.AppSpec{
					Envs: map[string]string{
						"FOO": "laksdjf",
					},
					Secrets: map[string]bundle.Secret{},
				},
				Params:      map[string]interface{}{},
				Connections: map[string]interface{}{},
			},
			want: map[string]string{},
			err:  errors.New("the jq query for environment variable FOO produced an error: function not defined: laksdjf/0"),
		},
		{
			name: "error on multiple values",
			b: &bundle.Bundle{
				AppSpec: &bundle.AppSpec{
					Envs: map[string]string{
						"FOO": ".params.array[]",
					},
					Secrets: map[string]bundle.Secret{},
				},
				Params: map[string]interface{}{
					"properties": map[string]interface{}{
						"array": map[string]interface{}{
							"type":     "array",
							"minItems": 2,
							"items": map[string]interface{}{
								"type": "integer",
							},
						},
					},
				},
				Connections: map[string]interface{}{},
			},
			want: map[string]string{},
			err:  errors.New("the jq query for environment variable FOO produced multiple values, which isn't supported"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.b.LintEnvs()
			if tc.err != nil {
				if err == nil {
					t.Errorf("expected an error, got nil")
				} else if tc.err.Error() != err.Error() {
					t.Errorf("got %v, want %v", err.Error(), tc.err.Error())
				}
			} else if err != nil {
				t.Fatalf("%d, unexpected error", err)
			}

			if len(got) != len(tc.want) {
				t.Errorf("got %v, want %v", len(got), len(tc.want))
			}
			for key, wantValue := range tc.want {
				gotValue, ok := got[key]
				if !ok {
					t.Errorf("got %v, want %v", got, tc.want)
				}
				if gotValue != wantValue {
					t.Errorf("got %v, want %v", gotValue, wantValue)
				}
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
						Path:        "testdata/lintmodule",
						Provisioner: "opentofu",
					}},
					Params: map[string]interface{}{
						"properties": map[string]interface{}{
							"foo": map[string]interface{}{},
							"bar": map[string]interface{}{},
						},
					},
					Connections: map[string]interface{}{},
					Artifacts:   map[string]interface{}{},
					UI:          map[string]interface{}{},
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
					Params: map[string]interface{}{
						"properties": map[string]interface{}{
							"foo": map[string]interface{}{},
						},
					},
					Connections: map[string]interface{}{
						"properties": map[string]interface{}{
							"bar": map[string]interface{}{},
						},
					},
					Artifacts: map[string]interface{}{},
					UI:        map[string]interface{}{},
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
					Params: map[string]interface{}{
						"properties": map[string]interface{}{
							"foo": map[string]interface{}{},
						},
					},
					Connections: map[string]interface{}{},
					Artifacts:   map[string]interface{}{},
					UI:          map[string]interface{}{},
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
					Params: map[string]interface{}{
						"properties": map[string]interface{}{
							"foo": map[string]interface{}{},
							"bar": map[string]interface{}{},
							"baz": map[string]interface{}{},
						},
					},
					Connections: map[string]interface{}{},
					Artifacts:   map[string]interface{}{},
					UI:          map[string]interface{}{},
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
				Params: map[string]interface{}{
					"required": []interface{}{"foo"},
					"properties": map[string]interface{}{
						"foo": map[string]interface{}{
							"type": "string",
						},
					},
				},
				Connections: map[string]interface{}{},
				Artifacts:   map[string]interface{}{},
				UI:          map[string]interface{}{},
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
				Params: map[string]interface{}{
					"required": []interface{}{"bar"},
					"properties": map[string]interface{}{
						"foo": map[string]interface{}{
							"type": "string",
						},
					},
				},
				Connections: map[string]interface{}{},
				Artifacts:   map[string]interface{}{},
				UI:          map[string]interface{}{},
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
				Params: map[string]interface{}{
					"required": []interface{}{"foo"},
					"properties": map[string]interface{}{
						"foo": map[string]interface{}{
							"type":     "object",
							"required": []interface{}{"bar"},
							"properties": map[string]interface{}{
								"bar": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
				Connections: map[string]interface{}{},
				Artifacts:   map[string]interface{}{},
				UI:          map[string]interface{}{},
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
				Params: map[string]interface{}{
					"required": []interface{}{"foo"},
					"properties": map[string]interface{}{
						"foo": map[string]interface{}{
							"type":     "object",
							"required": []interface{}{"baz", "bar"},
							"properties": map[string]interface{}{
								"bar": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
				Connections: map[string]interface{}{},
				Artifacts:   map[string]interface{}{},
				UI:          map[string]interface{}{},
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

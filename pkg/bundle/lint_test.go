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
				Access:      "private",
				Schema:      "draft-07",
				Type:        "infrastructure",
				Params:      &bundle.Schema{},
				Connections: &bundle.Schema{},
				Artifacts:   &bundle.Schema{},
				UI:          map[string]interface{}{},
			},
			err: nil,
		},
		{
			name: "Invalid missing schema",
			bun: &bundle.Bundle{
				Name:        "example",
				Description: "description",
				Access:      "private",
				Type:        "infrastructure",
				Params:      &bundle.Schema{},
				Connections: &bundle.Schema{},
				Artifacts:   &bundle.Schema{},
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
				Params: &bundle.Schema{
					Properties: map[string]*bundle.Schema{
						"param": {},
					},
				},
				Connections: &bundle.Schema{
					Properties: map[string]*bundle.Schema{
						"connection": {},
					},
				},
			},
			err: nil,
		},
		{
			name: "Invalid Error",
			bun: &bundle.Bundle{
				Params: &bundle.Schema{
					Properties: map[string]*bundle.Schema{
						"database": {},
					},
				},
				Connections: &bundle.Schema{
					Properties: map[string]*bundle.Schema{
						"database": {},
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
				Params: &bundle.Schema{
					Properties: map[string]*bundle.Schema{
						"foo": {
							Type:  "string",
							Const: "bar",
						},
						"int": {
							Type:  "integer",
							Const: 4,
						},
					},
				},
				Connections: &bundle.Schema{
					Properties: map[string]*bundle.Schema{
						"connection1": {
							Type:  "string",
							Const: "whatever",
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
				Params:      &bundle.Schema{},
				Connections: &bundle.Schema{},
				Artifacts:   &bundle.Schema{},
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
				Params:      &bundle.Schema{},
				Connections: &bundle.Schema{},
				Artifacts:   &bundle.Schema{},
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
				Params: &bundle.Schema{
					Properties: map[string]*bundle.Schema{
						"array": {
							Type:     "array",
							MinItems: bundle.Ptr(uint64(2)),
							Items: &bundle.Schema{
								Type: "integer",
							},
						},
					},
				},
				Connections: &bundle.Schema{},
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
					Access:      "private",
					Schema:      "draft-07",
					Type:        "infrastructure",
					Steps: []bundle.Step{{
						Path:        "testdata/lintmodule",
						Provisioner: "opentofu",
					}},
					Params: &bundle.Schema{
						Properties: map[string]*bundle.Schema{
							"foo": {},
							"bar": {},
						},
					},
					Connections: &bundle.Schema{},
					Artifacts:   &bundle.Schema{},
					UI:          map[string]interface{}{},
				},
				err: nil,
			}, {
				name: "Valid param and connection",
				bun: &bundle.Bundle{
					Name:        "example",
					Description: "description",
					Access:      "private",
					Schema:      "draft-07",
					Type:        "infrastructure",
					Steps: []bundle.Step{{
						Path:        "testdata/lintmodule",
						Provisioner: "opentofu",
					}},
					Params: &bundle.Schema{
						Properties: map[string]*bundle.Schema{
							"foo": &bundle.Schema{},
						},
					},
					Connections: &bundle.Schema{
						Properties: map[string]*bundle.Schema{
							"bar": &bundle.Schema{},
						},
					},
					Artifacts: &bundle.Schema{},
					UI:        map[string]interface{}{},
				},
				err: nil,
			}, {
				name: "Invalid missing massdriver input",
				bun: &bundle.Bundle{
					Name:        "example",
					Description: "description",
					Access:      "private",
					Type:        "infrastructure",
					Steps: []bundle.Step{{
						Path:        "testdata/lintmodule",
						Provisioner: "opentofu",
					}},
					Params: &bundle.Schema{
						Properties: map[string]*bundle.Schema{
							"foo": &bundle.Schema{},
						},
					},
					Connections: &bundle.Schema{},
					Artifacts:   &bundle.Schema{},
					UI:          map[string]interface{}{},
				},
				err: errors.New(`missing inputs detected in step testdata/lintmodule:
	- input "bar" declared in provisioner but missing massdriver.yaml declaration
`),
			}, {
				name: "Invalid missing provisioner input",
				bun: &bundle.Bundle{
					Name:        "example",
					Description: "description",
					Access:      "private",
					Type:        "infrastructure",
					Steps: []bundle.Step{{
						Path:        "testdata/lintmodule",
						Provisioner: "opentofu",
					}},
					Params: &bundle.Schema{
						Properties: map[string]*bundle.Schema{
							"foo": &bundle.Schema{},
							"bar": &bundle.Schema{},
							"baz": &bundle.Schema{},
						},
					},
					Connections: &bundle.Schema{},
					Artifacts:   &bundle.Schema{},
					UI:          map[string]interface{}{},
				},
				err: errors.New(`missing inputs detected in step testdata/lintmodule:
	- input "baz" declared in massdriver.yaml but missing provisioner declaration
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
				Access:      "private",
				Schema:      "draft-07",
				Type:        "infrastructure",
				Params: &bundle.Schema{
					Required: []string{"foo"},
					Properties: map[string]*bundle.Schema{
						"foo": {
							Type: "string",
						},
					},
				},
				Connections: &bundle.Schema{},
				Artifacts:   &bundle.Schema{},
				UI:          map[string]interface{}{},
			},
			err: nil,
		},
		{
			name: "Invalid missing param",
			bun: &bundle.Bundle{
				Name:        "example",
				Description: "description",
				Access:      "private",
				Schema:      "draft-07",
				Type:        "infrastructure",
				Params: &bundle.Schema{
					Required: []string{"bar"},
					Properties: map[string]*bundle.Schema{
						"foo": {
							Type: "string",
						},
					},
				},
				Connections: &bundle.Schema{},
				Artifacts:   &bundle.Schema{},
				UI:          map[string]interface{}{},
			},
			err: errors.New("required parameter bar is not defined in properties"),
		},
		{
			name: "Nested valid test",
			bun: &bundle.Bundle{
				Name:        "example",
				Description: "description",
				Access:      "private",
				Schema:      "draft-07",
				Type:        "infrastructure",
				Params: &bundle.Schema{
					Required: []string{"foo"},
					Properties: map[string]*bundle.Schema{
						"foo": {
							Type:     "object",
							Required: []string{"bar"},
							Properties: map[string]*bundle.Schema{
								"bar": {
									Type: "string",
								},
							},
						},
					},
				},
				Connections: &bundle.Schema{},
				Artifacts:   &bundle.Schema{},
				UI:          map[string]interface{}{},
			},
			err: nil,
		},
		{
			name: "Nested invalid test",
			bun: &bundle.Bundle{
				Name:        "example",
				Description: "description",
				Access:      "private",
				Schema:      "draft-07",
				Type:        "infrastructure",
				Params: &bundle.Schema{
					Required: []string{"foo"},
					Properties: map[string]*bundle.Schema{
						"foo": {
							Type:     "object",
							Required: []string{"baz", "bar"},
							Properties: map[string]*bundle.Schema{
								"bar": {
									Type: "string",
								},
							},
						},
					},
				},
				Connections: &bundle.Schema{},
				Artifacts:   &bundle.Schema{},
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

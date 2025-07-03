package bundle_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"

	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/client"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/config"
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
				Params:      map[string]any{"properties": map[string]any{}},
				Connections: map[string]any{"properties": map[string]any{}},
				Artifacts:   map[string]any{"properties": map[string]any{}},
				UI:          map[string]any{"properties": map[string]any{}},
			},
			err: nil,
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
			err: errors.New(`missing property 'schema'`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bundleSchema, err := ioutil.ReadFile("testdata/lint/schema/bundle.json")
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

			err = tc.bun.LintSchema(&mdClient)
			if tc.err != nil {
				if err == nil {
					t.Errorf("expected an error, got nil")
				} else if !strings.Contains(err.Error(), tc.err.Error()) {
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
				err: nil,
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
				err: errors.New(`missing inputs detected in step testdata/lint/module:
	- input "bar" declared in IaC but missing massdriver.yaml declaration
`),
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
				err: errors.New(`missing inputs detected in step testdata/lint/module:
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

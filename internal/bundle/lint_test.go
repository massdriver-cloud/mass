package bundle_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/massdriver-cloud/mass/internal/bundle"
)

func TestLintSchema(t *testing.T) {
	type test struct {
		name string
		bun  *bundle.Bundle
		err  error
	}

	// TODO: this is currently failing because we are using b.Conf to get actualy files to pass, but this
	// is mocking w/o it... fix this code to use the actual massdriver.yaml
	tests := []test{
		{
			name: "Valid pass",
			bun: &bundle.Bundle{
				Name:        "example",
				Description: "description",
				Access:      "private",
				Schema:      "draft-07",
				Type:        "infrastructure",
				Params:      map[string]interface{}{},
				Connections: map[string]interface{}{},
				Artifacts:   map[string]interface{}{},
				UI:          map[string]interface{}{},
				Steps:       []bundle.Step{},
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
				Params:      map[string]interface{}{},
				Connections: map[string]interface{}{},
				Artifacts:   map[string]interface{}{},
				UI:          map[string]interface{}{},
				Steps:       []bundle.Step{},
			},
			err: errors.New(`massdriver.yaml has schema violations:
	- (root): Must validate one and only one schema (oneOf)
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

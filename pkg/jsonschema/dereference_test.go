package jsonschema_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/jsonschema"
	"github.com/massdriver-cloud/mass/pkg/restclient"
)

func TestDereference(t *testing.T) {
	wd, _ := os.Getwd()

	type TestCase struct {
		Name                string
		Input               interface{}
		Expected            interface{}
		ExpectedErrorSuffix string
		Cwd                 string
	}

	cases := []TestCase{
		{
			Name:  "Dereferences a $ref",
			Input: jsonDecode(`{"$ref": "./testdata/artifacts/aws-example.json"}`),
			Expected: map[string]string{
				"id": "fake-schema-id",
			},
		},
		{
			Name:  "Dereferences a $ref alongside arbitrary values",
			Input: jsonDecode(`{"foo": true, "bar": {}, "$ref": "./testdata/artifacts/aws-example.json"}`),
			Expected: map[string]interface{}{
				"foo": true,
				"bar": map[string]interface{}{},
				"id":  "fake-schema-id",
			},
		},
		{
			Name:  "Dereferences a nested $ref",
			Input: jsonDecode(`{"key": {"$ref": "./testdata/artifacts/aws-example.json"}}`),
			Expected: map[string]map[string]string{
				"key": {
					"id": "fake-schema-id",
				},
			},
		},
		{
			Name:  "Does not dereference fragment (#) refs",
			Input: jsonDecode(`{"$ref": "#/its-in-this-file"}`),
			Expected: map[string]string{
				"$ref": "#/its-in-this-file",
			},
		},
		{
			Name:  "Dereferences $refs in a list",
			Input: jsonDecode(`{"list": ["string", {"$ref": "./testdata/artifacts/aws-example.json"}]}`),
			Expected: map[string]interface{}{
				"list": []interface{}{
					"string",
					map[string]interface{}{
						"id": "fake-schema-id",
					},
				},
			},
		},
		{
			Name:  "Dereferences a $ref deterministically (keys outside of ref always win)",
			Input: jsonDecode(`{"conflictingKey": "not-from-ref", "$ref": "./testdata/artifacts/conflicting-keys.json"}`),
			Expected: map[string]string{
				"conflictingKey": "not-from-ref",
				"nonConflictKey": "from-ref",
			},
		},
		{
			Name:  "Dereferences a $ref recursively",
			Input: jsonDecode(`{"$ref": "./testdata/artifacts/ref-aws-example.json"}`),
			Expected: map[string]map[string]string{
				"properties": {
					"id": "fake-schema-id",
				},
			},
		},
		{
			Name:  "Dereferences a $ref recursively",
			Input: jsonDecode(`{"$ref": "./testdata/artifacts/ref-lower-dir-aws-example.json"}`),
			Expected: map[string]map[string]string{
				"properties": {
					"id": "fake-schema-id",
				},
			},
		},
		{
			Name:                "Reports not found when $ref is not found",
			Input:               jsonDecode(`{"$ref": "./testdata/no-type.json"}`),
			ExpectedErrorSuffix: fmt.Sprintf("open %s: no such file or directory", path.Join(wd, "testdata/no-type.json")),
		},
		{
			Name:  "Dereferences remote (massdriver) ref",
			Input: jsonDecode(`{"$ref": "massdriver/test-schema"}`),
			Expected: map[string]string{
				"foo": "bar",
			},
		},
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				urlPath := r.URL.Path
				switch urlPath {
				case "/artifact-definitions/massdriver/test-schema":
					if _, err := w.Write([]byte(`{"foo":"bar"}`)); err != nil {
						t.Fatalf("Failed to write response: %v", err)
					}
				default:
					t.Fatalf("unknown schema: %v", urlPath)
				}
			}))
			defer testServer.Close()

			c := restclient.NewClient()
			c.WithBaseURL(testServer.URL)

			opts := jsonschema.DereferenceOptions{
				Client: c,
				Cwd:    ".",
			}

			got, gotErr := jsonschema.Dereference(test.Input, opts)

			if test.ExpectedErrorSuffix != "" {
				if !strings.HasSuffix(gotErr.Error(), test.ExpectedErrorSuffix) {
					t.Errorf("got %v, want %v", gotErr.Error(), test.ExpectedErrorSuffix)
				}
			} else {
				if fmt.Sprint(got) != fmt.Sprint(test.Expected) {
					t.Errorf("got %v, want %v", got, test.Expected)
				}
			}
		})
	}

	// Easier to test HTTP refs separately
	t.Run("HTTP Refs", func(t *testing.T) {
		var recursivePtr *string
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			urlPath := r.URL.Path
			switch urlPath {
			case "/recursive":
				if _, err := w.Write([]byte(*recursivePtr)); err != nil {
					t.Fatalf("Failed to write response: %v", err)
				}
				fmt.Println("in recursive")
			case "/endpoint":
				if _, err := w.Write([]byte(`{"foo":"bar"}`)); err != nil {
					t.Fatalf("Failed to write response: %v", err)
				}
				fmt.Println("in endpoint")
			default:
				w.WriteHeader(http.StatusNotFound)
				_, err := w.Write([]byte(`404 - not found`))
				if err != nil {
					t.Fatalf("Failed to write response: %v", err)
				}
			}
		}))
		defer testServer.Close()

		c := restclient.NewClient().WithBaseURL(testServer.URL)

		recursive := fmt.Sprintf(`{"baz":{"$ref":"%s/endpoint"}}`, testServer.URL)
		recursivePtr = &recursive

		input := jsonDecode(fmt.Sprintf(`{"$ref":"%s/recursive"}`, testServer.URL))

		opts := jsonschema.DereferenceOptions{
			Client: c,
			Cwd:    ".",
		}
		got, _ := jsonschema.Dereference(input, opts)
		expected := map[string]interface{}{
			"baz": map[string]string{
				"foo": "bar",
			},
		}

		if fmt.Sprint(got) != fmt.Sprint(expected) {
			t.Errorf("got %v, want %v", got, expected)
		}

		input = jsonDecode(fmt.Sprintf(`{"$ref":"%s/not-found"}`, testServer.URL))

		opts = jsonschema.DereferenceOptions{
			Client: c,
			Cwd:    ".",
		}
		_, gotErr := jsonschema.Dereference(input, opts)
		expectedErrPrefix := "received non-200 response getting ref 404 Not Found"

		if !strings.HasPrefix(gotErr.Error(), expectedErrPrefix) {
			t.Errorf("got %v, want %v", gotErr.Error(), expectedErrPrefix)
		}
	})
}

func jsonDecode(data string) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		panic(err)
	}
	return result
}

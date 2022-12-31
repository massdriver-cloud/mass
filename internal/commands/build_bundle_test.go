package commands_test

import (
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"github.com/massdriver-cloud/mass/internal/bundle"
	"github.com/massdriver-cloud/mass/internal/commands"
	"github.com/massdriver-cloud/mass/internal/mockfilesystem"
	"github.com/massdriver-cloud/mass/internal/restclient"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

var expectedSchemaContents = map[string][]byte{
	"ui-schema.json": []byte(`{
    "ui:order": [
        "resource_name",
        "*"
    ]
}
`),
	"params-schema.json": []byte(`{
    "$id": "https://schemas.massdriver.cloud/schemas/bundles/draft-node/schema-params.json",
    "$schema": "http://json-schema.org/draft-07/schema",
    "description": "A resource that can be used to visually design architecture without provisioning real infrastructure.",
    "examples": [
        {
            "__name": "Network",
            "resource_type": "Network"
        }
    ],
    "properties": {
        "resource_name": {
            "$md.immutable": true,
            "description": "An immutable name field",
            "title": "Resource Name",
            "type": "string"
        }
    },
    "title": "draft-node"
}
`),
	"connections-schema.json": []byte(`{
    "$id": "https://schemas.massdriver.cloud/schemas/bundles/draft-node/schema-connections.json",
    "$schema": "http://json-schema.org/draft-07/schema",
    "description": "A resource that can be used to visually design architecture without provisioning real infrastructure.",
    "properties": {
        "draft_node_foo": {
            "properties": {
                "foo": "bar"
            },
            "type": "object"
        }
    },
    "required": [
        "draft_node_foo"
    ],
    "title": "draft-node"
}
`),
	"artifacts-schema.json": []byte(`{
    "$id": "https://schemas.massdriver.cloud/schemas/bundles/draft-node/schema-artifacts.json",
    "$schema": "http://json-schema.org/draft-07/schema",
    "description": "A resource that can be used to visually design architecture without provisioning real infrastructure.",
    "properties": {
        "draft_node": {
            "properties": {
                "foo": "bar"
            },
            "type": "object"
        }
    },
    "required": [
        "draft_node"
    ],
    "title": "draft-node"
}
`),
}

var expectedTFContent = map[string][]byte{
	"_connections_variables.tf.json": []byte(`{
    "variable": {
        "draft_node_foo": {
            "type": "any",
            "default": null
        }
    }
}
`),
	"_md_variables.tf.json": []byte(`{
    "variable": {
        "md_metadata": {
            "type": "any",
            "default": null
        }
    }
}
`),
	"_params_variables.tf.json": []byte(`{
    "variable": {
        "resource_name": {
            "type": "string",
            "default": null
        }
    }
}
`),
}

func TestBundleBuildSchemas(t *testing.T) {
	writeDir := "."
	fs := afero.NewMemMapFs()
	err := mockfilesystem.SetupBundle(writeDir, fs)

	if err != nil {
		t.Fatal(err)
	}

	file, err := afero.ReadFile(fs, path.Join(writeDir, "massdriver.yaml"))

	if err != nil {
		t.Fatal(err)
	}

	unmarshalledBundle := &bundle.Bundle{}
	err = yaml.Unmarshal(file, unmarshalledBundle)

	if err != nil {
		t.Fatal(err)
	}

	testServer := setupMockServer(t)

	defer testServer.Close()

	c := restclient.NewClient()
	c.WithBaseURL(testServer.URL)

	commands.BuildBundle(writeDir, unmarshalledBundle, c, fs)

	for fileName, expectedFileContent := range expectedSchemaContents {
		gotContent, _ := afero.ReadFile(fs, path.Join(writeDir, fileName))
		if string(gotContent) != string(expectedFileContent) {
			t.Errorf("Expected file content for %s to be %s but got %s", fileName, string(expectedFileContent), string(gotContent))
		}
	}
}

func TestBundleBuildTFVars(t *testing.T) {
	writeDir := "."
	fs := afero.NewMemMapFs()
	err := mockfilesystem.SetupBundle(writeDir, fs)

	if err != nil {
		t.Fatal(err)
	}

	file, err := afero.ReadFile(fs, path.Join(writeDir, "massdriver.yaml"))

	if err != nil {
		t.Fatal(err)
	}

	unmarshalledBundle := &bundle.Bundle{}
	err = yaml.Unmarshal(file, unmarshalledBundle)

	if err != nil {
		t.Fatal(err)
	}

	testServer := setupMockServer(t)

	defer testServer.Close()

	c := restclient.NewClient()
	c.WithBaseURL(testServer.URL)

	commands.BuildBundle(writeDir, unmarshalledBundle, c, fs)

	for fileName, expectedContent := range expectedTFContent {
		gotContent, _ := afero.ReadFile(fs, path.Join(writeDir, "src", fileName))
		if string(gotContent) != string(expectedContent) {
			t.Errorf("Expected file content for %s to be %s but got %s", fileName, string(expectedContent), string(gotContent))
		}
	}
}

func setupMockServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path
		switch urlPath {
		case "/artifact-definitions/massdriver/draft-node":
			if _, err := w.Write([]byte(`{"type": "object", "properties": {"foo": "bar"}}`)); err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}
		default:
			t.Fatalf("unknown schema: %v", urlPath)
		}
	}))
}

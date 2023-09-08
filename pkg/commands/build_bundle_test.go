package commands_test

import (
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/commands"
	"github.com/massdriver-cloud/mass/pkg/mockfilesystem"
	"github.com/massdriver-cloud/mass/pkg/restclient"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

var expectedSchemaContents = map[string][]byte{
	"schema-ui.json": []byte(`{
    "ui:order": [
        "resource_name",
        "*"
    ]
}
`),
	"schema-params.json": []byte(`{
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
        "foo": {
            "description": "A map of Foos",
            "properties": {
                "bar": {
                    "default": 1,
                    "description": "Testing numbers",
                    "title": "A whole number",
                    "type": "integer"
                },
                "qux": {
                    "description": "Testing numbers",
                    "minimum": 2,
                    "title": "A whole number that is not required",
                    "type": "integer"
                }
            },
            "required": [
                "bar"
            ],
            "title": "Foo",
            "type": "object"
        },
        "resource_name": {
            "$md.immutable": true,
            "description": "An immutable name field",
            "title": "Resource Name",
            "type": "string"
        },
        "resource_type": {
            "description": "The type of resource",
            "title": "Resource Type",
            "type": "string"
        }
    },
    "required": [
        "resource_type"
    ],
    "title": "draft-node"
}
`),
	"schema-connections.json": []byte(`{
    "$id": "https://schemas.massdriver.cloud/schemas/bundles/draft-node/schema-connections.json",
    "$schema": "http://json-schema.org/draft-07/schema",
    "description": "A resource that can be used to visually design architecture without provisioning real infrastructure.",
    "properties": {
        "draft_node_foo": {
            "properties": {
                "foo": {
                    "properties": {
                        "infrastructure": {
                            "properties": {
                                "arn": {
                                    "type": "string"
                                }
                            },
                            "type": "object"
                        }
                    },
                    "type": "object"
                }
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
	"schema-artifacts.json": []byte(`{
    "$id": "https://schemas.massdriver.cloud/schemas/bundles/draft-node/schema-artifacts.json",
    "$schema": "http://json-schema.org/draft-07/schema",
    "description": "A resource that can be used to visually design architecture without provisioning real infrastructure.",
    "properties": {
        "draft_node": {
            "properties": {
                "foo": {
                    "properties": {
                        "infrastructure": {
                            "properties": {
                                "arn": {
                                    "type": "string"
                                }
                            },
                            "type": "object"
                        }
                    },
                    "type": "object"
                }
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
        "foo": {
            "type": "any",
            "default": null
        },
        "resource_name": {
            "type": "string",
            "default": null
        },
        "resource_type": {
            "type": "string",
            "default": null
        }
    }
}
`),
	"_params.auto.tfvars.json": []byte(`{
    "foo": {
        "bar": 1,
        "qux": 2
    },
    "md_metadata": {
        "default_tags": {
            "md-manifest": "draft-node",
            "md-package": "local-dev-draft-node-000",
            "md-project": "local",
            "md-target": "dev"
        },
        "deployment": {
            "id": "local-dev-id"
        },
        "name_prefix": "local-dev-draft-node-000",
        "observability": {
            "alarm_webhook_url": "https://placeholder.com"
        }
    },
    "resource_name": "REPLACE ME",
    "resource_type": "Network"
}`),
	"_connections.auto.tfvars.json": []byte(`{
    "draft_node_foo": {
        "foo": {
            "infrastructure": {
                "arn": "REPLACE ME"
            }
        }
    }
}`),
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
	c.WithAPIKey("dummy")

	err = commands.BuildBundle(writeDir, unmarshalledBundle, c, fs)

	if err != nil {
		t.Fatal(err)
	}

	for fileName, expectedFileContent := range expectedSchemaContents {
		gotContent, readFileErr := afero.ReadFile(fs, path.Join(writeDir, fileName))
		if readFileErr != nil {
			t.Fatal(readFileErr)
		}
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
	c.WithAPIKey("dummy")

	err = commands.BuildBundle(writeDir, unmarshalledBundle, c, fs)

	if err != nil {
		t.Fatal(err)
	}

	for fileName, expectedContent := range expectedTFContent {
		gotContent, readFileErr := afero.ReadFile(fs, path.Join(writeDir, "src", fileName))
		if readFileErr != nil {
			t.Fatal(readFileErr)
		}
		if string(gotContent) != string(expectedContent) {
			t.Errorf("Expected file content for %s to be %s but got %s", fileName, string(expectedContent), string(gotContent))
		}
	}
}

var draftNodeAD = []byte(`{"type": "object", "properties": {"foo": {"type": "object", "properties": {"infrastructure": {"type": "object", "properties": {"arn": {"type": "string"}}}}}}}`)

func setupMockServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path
		switch urlPath {
		case "/artifact-definitions/massdriver/draft-node":
			if _, err := w.Write(draftNodeAD); err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}
		default:
			t.Fatalf("unknown schema: %v", urlPath)
		}
	}))
}

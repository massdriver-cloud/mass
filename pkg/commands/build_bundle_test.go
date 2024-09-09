package commands_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	"github.com/massdriver-cloud/mass/pkg/commands"
	"github.com/massdriver-cloud/mass/pkg/mockfilesystem"
	"github.com/massdriver-cloud/mass/pkg/restclient"
	"sigs.k8s.io/yaml"
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
	"_massdriver_variables.tf": []byte(`// Auto-generated variable declarations from massdriver.yaml
variable "draft_node_foo" {
  type = object({
    foo = optional(object({
      infrastructure = optional(object({
        arn = optional(string)
      }))
    }))
  })
}
variable "foo" {
  type = object({
    bar = number
    qux = optional(number)
  })
  default = null
}
variable "md_metadata" {
  type = object({
    default_tags = object({
      managed-by  = string
      md-manifest = string
      md-package  = string
      md-project  = string
      md-target   = string
    })
    deployment = object({
      id = string
    })
    name_prefix = string
    observability = object({
      alarm_webhook_url = string
    })
    package = object({
      created_at             = string
      deployment_enqueued_at = string
      previous_status        = string
      updated_at             = string
    })
    target = object({
      contact_email = string
    })
  })
}
variable "resource_name" {
  type    = string
  default = null
}
variable "resource_type" {
  type = string
}
`),
}

func TestBundleBuildSchemas(t *testing.T) {
	testDir := t.TempDir()
	err := mockfilesystem.SetupBundle(testDir)

	if err != nil {
		t.Fatal(err)
	}

	file, err := os.ReadFile(path.Join(testDir, "massdriver.yaml"))

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

	err = commands.BuildBundle(testDir, unmarshalledBundle, c)

	if err != nil {
		t.Fatal(err)
	}

	for fileName, expectedFileContent := range expectedSchemaContents {
		gotContent, readFileErr := os.ReadFile(path.Join(testDir, fileName))
		if readFileErr != nil {
			t.Fatal(readFileErr)
		}
		if string(gotContent) != string(expectedFileContent) {
			t.Errorf("Expected file content for %s to be %s but got %s", fileName, string(expectedFileContent), string(gotContent))
		}
	}
}

func TestBundleBuildTFVars(t *testing.T) {
	testDir := t.TempDir()
	err := mockfilesystem.SetupBundle(testDir)

	if err != nil {
		t.Fatal(err)
	}

	file, err := os.ReadFile(path.Join(testDir, "massdriver.yaml"))

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

	err = commands.BuildBundle(testDir, unmarshalledBundle, c)

	if err != nil {
		t.Fatal(err)
	}

	for fileName, expectedContent := range expectedTFContent {
		gotContent, readFileErr := os.ReadFile(path.Join(testDir, "src", fileName))
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

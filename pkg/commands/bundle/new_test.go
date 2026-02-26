package bundle_test

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/massdriver-cloud/mass/pkg/bundle"
	cmdbundle "github.com/massdriver-cloud/mass/pkg/commands/bundle"
	"github.com/massdriver-cloud/mass/pkg/mockfilesystem"
	"github.com/massdriver-cloud/mass/pkg/templates"
	"sigs.k8s.io/yaml"
)

func TestCopyFilesFromTemplateToCurrentDirectory(t *testing.T) {
	testDir := t.TempDir()
	rootTemplateDir := path.Join(testDir, "/home/md-cloud")

	err := mockfilesystem.SetupBundleTemplate(rootTemplateDir)
	checkErr(err, t)

	tmpl := &templates.Templates{Path: rootTemplateDir}
	data := mockTemplateData(testDir)

	err = cmdbundle.RunNew(tmpl, data)
	checkErr(err, t)

	wantTopLevel := []string{"home", "massdriver.yaml", "src"}
	if errorString, assertion := mockfilesystem.AssertDirectoryContents(testDir, wantTopLevel); !assertion {
		t.Errorf("%s", errorString)
	}

	wantSecondLevel := []string{"main.tf"}
	if errorString, assertion := mockfilesystem.AssertDirectoryContents(path.Join(testDir, "src"), wantSecondLevel); !assertion {
		t.Errorf("%s", errorString)
	}
}

func TestCopyFilesFromTemplateToNonExistentDirectory(t *testing.T) {
	testDir := t.TempDir()
	rootTemplateDir := path.Join(testDir, "/home/md-cloud")
	writePath := path.Join(testDir, "./bundles/aws-sqs-queue")

	err := mockfilesystem.SetupBundleTemplate(rootTemplateDir)
	checkErr(err, t)

	tmpl := &templates.Templates{Path: rootTemplateDir}
	data := mockTemplateData(writePath)

	err = cmdbundle.RunNew(tmpl, data)
	checkErr(err, t)

	wantTopLevel := []string{"massdriver.yaml", "src"}
	if errorString, assertion := mockfilesystem.AssertDirectoryContents(writePath, wantTopLevel); !assertion {
		t.Errorf("%s", errorString)
	}

	wantSecondLevel := []string{"main.tf"}
	if errorString, assertion := mockfilesystem.AssertDirectoryContents(path.Join(writePath, "src"), wantSecondLevel); !assertion {
		t.Errorf("%s", errorString)
	}
}

func TestTemplateRender(t *testing.T) {
	testDir := t.TempDir()
	rootTemplateDir := path.Join(testDir, "/home/md-cloud")

	err := mockfilesystem.SetupBundleTemplate(rootTemplateDir)
	checkErr(err, t)

	tmpl := &templates.Templates{Path: rootTemplateDir}
	data := mockTemplateData(testDir)

	err = tmpl.Render(data)
	checkErr(err, t)

	renderedTemplate, err := os.ReadFile(path.Join(testDir, "massdriver.yaml"))
	fmt.Println(string(renderedTemplate))
	checkErr(err, t)

	got := &bundle.Bundle{}
	err = yaml.Unmarshal(renderedTemplate, got)
	checkErr(err, t)

	wantConnections := map[string]any{
		"properties": map[string]any{
			"aws_authentication": map[string]any{"$ref": "massdriver/aws-iam-role"},
			"dynamo":             map[string]any{"$ref": "massdriver/aws-dynamodb-table"},
		},
		"required": []any{"aws_authentication", "dynamo"},
	}

	if got.Name != data.Name {
		t.Errorf("Expected rendered template's name field to be %s but got %s", data.Name, got.Name)
	}

	if !reflect.DeepEqual(got.Connections, wantConnections) {
		t.Errorf("Expected rendered template's connections field to be %v but got %v", wantConnections, got.Connections)
	}
}

func mockTemplateData(writePath string) *templates.TemplateData {
	return &templates.TemplateData{
		OutputDir:         writePath,
		Type:              "infrastructure",
		TemplateName:      "opentofu",
		Name:              "aws-dynamodb",
		Description:       "whatever",
		Connections:       []templates.Connection{{ArtifactDefinition: "massdriver/aws-dynamodb-table", Name: "dynamo"}},
		CloudAbbreviation: "aws",
		RepoName:          "massdriver-cloud/bundle-templates",
		RepoNameEncoded:   "massdriver-cloud/bundle-templates",
	}
}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}
